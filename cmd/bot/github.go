package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GitHub API 结构
type gitHubRelease struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type gitCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Author struct {
			Name string `json:"name"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
}

// githubToken 全局 GitHub Token（可选）
var githubToken string

// httpClient 全局 HTTP 客户端（复用连接）
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
	},
}

// setGitHubHeaders 设置 GitHub API 请求头
func setGitHubHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "newrelease")
	if githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+githubToken)
	}
}

// checkRateLimit 检查并记录 GitHub API Rate Limit
func checkRateLimit(resp *http.Response) {
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	limit := resp.Header.Get("X-RateLimit-Limit")
	
	if remaining != "" && limit != "" {
		remainingNum, _ := strconv.Atoi(remaining)
		limitNum, _ := strconv.Atoi(limit)
		
		if remainingNum < 100 {
			log.Printf("⚠️  GitHub API rate limit LOW: %d/%d remaining", remainingNum, limitNum)
		}
		if remainingNum < 10 {
			log.Printf("🚨 GitHub API rate limit CRITICAL: %d/%d remaining", remainingNum, limitNum)
		}
	}
}

// getLatestRelease 获取最新 Release
func getLatestRelease(client *http.Client, repo string) (*gitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	setGitHubHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查 Rate Limit
	checkRateLimit(resp)

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("Repository %s not found (404). It might be private or have a typo.", repo)
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d from GitHub", resp.StatusCode)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// getLatestCommit 获取最新 Commit
func getLatestCommit(client *http.Client, repo, branch string) (*gitCommit, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/commits?sha=%s&per_page=1", repo, url.QueryEscape(branch))
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	setGitHubHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查 Rate Limit
	checkRateLimit(resp)

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("Repository %s or branch %s not found (404).", repo, branch)
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d from GitHub", resp.StatusCode)
	}

	var commits []gitCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, nil
	}
	return &commits[0], nil
}

// scheduledChecker 定时检查器
func scheduledChecker(tg *telegramClient, adminID int64) {
	time.Sleep(initialDelay)

	for {
		log.Printf("Running scheduled check for new releases...")
		configs, err := loadConfigs()
		if err != nil {
			log.Printf("Failed to load configs: %v", err)
		} else if len(configs) == 0 {
			log.Printf("No configurations found. Skipping check.")
		} else {
			// 批量保存标志
			configChanged := false

			for i := range configs {
				if configs[i].MonitorRelease {
					release, err := getLatestRelease(httpClient, configs[i].Repo)
					if err != nil {
						log.Printf("Error fetching GitHub release for %s: %v", configs[i].Repo, err)
					} else if release != nil {
						if configs[i].LastReleaseID == nil || *configs[i].LastReleaseID != release.ID {
							log.Printf("New release found for %s: %s", configs[i].Repo, release.Name)
							if configs[i].LastReleaseID != nil {
								name := release.Name
								if name == "" {
									name = release.TagName
								}
								name = escapeMarkdown(name)
								messageText := fmt.Sprintf(
									releaseMessageTmpl,
									configs[i].Repo,
									release.TagName,
									release.HTMLURL,
								)
								targetID := configs[i].ChannelID
								if targetID == 0 {
									targetID = adminID
								}
								if _, err := tg.sendMessage(targetID, messageText, telegramParseModeMarkdown, true, ""); err != nil {
									log.Printf("Failed to send message to %d: %v", targetID, err)
								}
							}

							latestID := release.ID
							configs[i].LastReleaseID = &latestID
							configChanged = true
						}
					}
				}

				if configs[i].MonitorCommit {
					branch := configs[i].Branch
					if branch == "" {
						branch = defaultBranch
					}
					commit, err := getLatestCommit(httpClient, configs[i].Repo, branch)
					if err != nil {
						log.Printf("Error fetching GitHub commit for %s (branch: %s): %v", configs[i].Repo, branch, err)
					} else if commit != nil {
						if configs[i].LastCommitSHA == nil || *configs[i].LastCommitSHA != commit.SHA {
							if configs[i].LastCommitSHA != nil {
								// 获取完整的 commit 消息
								message := strings.TrimSpace(commit.Commit.Message)
								if message == "" {
									message = commit.SHA
								}
		
							author := strings.TrimSpace(commit.Commit.Author.Name)
							if author == "" && commit.Author != nil {
								author = commit.Author.Login
							}
							if author == "" {
								author = "未知作者"
							}
							author = escapeMarkdown(author)

							// 提取仓库名（只要 repo 部分，不要 owner）
							repoParts := strings.Split(configs[i].Repo, "/")
							repoName := configs[i].Repo
							if len(repoParts) == 2 {
								repoName = repoParts[1]
							}

							messageText := fmt.Sprintf(
								commitMessageTmpl,
								repoName,
								branch,
								message,
								commit.HTMLURL,
							)
								targetID := configs[i].ChannelID
								if targetID == 0 {
									targetID = adminID
								}
								if _, err := tg.sendMessage(targetID, messageText, telegramParseModeMarkdown, true, ""); err != nil {
									log.Printf("Failed to send commit message to %d: %v", targetID, err)
								}
							}

							latestSHA := commit.SHA
							configs[i].LastCommitSHA = &latestSHA
							configChanged = true
						}
					}
				}

				time.Sleep(repoCheckDelay)
			}

			// 批量保存：只在有变化时保存一次
			if configChanged {
				if err := saveConfigs(configs); err != nil {
					log.Printf("Failed to save configs: %v", err)
				} else {
					log.Printf("Configurations updated and saved successfully")
				}
			}
		}

		log.Printf("Check finished. Waiting for %s.", checkInterval)
		time.Sleep(checkInterval)
	}
}
