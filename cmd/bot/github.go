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
		Message string `json:"message"`
	} `json:"commit"`
}

type gitHubRepo struct {
	DefaultBranch string `json:"default_branch"`
}

// githubToken 全局 GitHub Token（可选）
var githubToken string

// httpClient 全局 HTTP 客户端（复用连接）
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
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
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	log.Printf("🐙 GitHub API: GET %s", endpoint)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	setGitHubHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ GitHub API error for %s: %v", repo, err)
		return nil, err
	}
	defer resp.Body.Close()

	checkRateLimit(resp)

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("🔍 No releases found for %s", repo)
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("❌ GitHub API returned status %d for %s", resp.StatusCode, repo)
		return nil, fmt.Errorf("unexpected status %d from GitHub", resp.StatusCode)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Printf("❌ Failed to decode release response for %s: %v", repo, err)
		return nil, err
	}
	log.Printf("✔️ Found release for %s: %s (ID: %d)", repo, release.TagName, release.ID)
	return &release, nil
}

// getLatestCommit 获取最新 Commit
func getLatestCommit(client *http.Client, repo, branch string) (*gitCommit, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/commits?sha=%s&per_page=1", repo, url.QueryEscape(branch))
	log.Printf("🐙 GitHub API: GET %s", endpoint)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	setGitHubHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ GitHub API error for %s:%s: %v", repo, branch, err)
		return nil, err
	}
	defer resp.Body.Close()

	checkRateLimit(resp)

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("🔍 No commits found for %s:%s", repo, branch)
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("❌ GitHub API returned status %d for %s:%s", resp.StatusCode, repo, branch)
		return nil, fmt.Errorf("unexpected status %d from GitHub", resp.StatusCode)
	}

	var commits []gitCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		log.Printf("❌ Failed to decode commit response for %s:%s: %v", repo, branch, err)
		return nil, err
	}
	if len(commits) == 0 {
		log.Printf("🔍 Empty commits array for %s:%s", repo, branch)
		return nil, nil
	}
	log.Printf("✔️ Found commit for %s:%s: %.7s", repo, branch, commits[0].SHA)
	return &commits[0], nil
}

// getRepoDefaultBranch 获取仓库的默认分支
func getRepoDefaultBranch(client *http.Client, repo string) (string, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s", repo)
	log.Printf("🐙 GitHub API: GET %s (fetching default branch)", endpoint)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	setGitHubHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Failed to get repo info for %s: %v", repo, err)
		return "", err
	}
	defer resp.Body.Close()

	checkRateLimit(resp)

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ GitHub API returned status %d for repo %s", resp.StatusCode, repo)
		return "", fmt.Errorf("failed to get repo info: status %d", resp.StatusCode)
	}

	var repoInfo gitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		log.Printf("❌ Failed to decode repo info for %s: %v", repo, err)
		return "", err
	}
	log.Printf("✔️ Default branch for %s: %s", repo, repoInfo.DefaultBranch)
	return repoInfo.DefaultBranch, nil
}

// scheduledChecker 定时检查器
func scheduledChecker(tg *telegramClient, adminID int64) {
	time.Sleep(initialDelay)

	for {
		log.Printf("Running scheduled check...")
		configs, err := loadConfigs()
		if err != nil {
			log.Printf("Failed to load configs: %v", err)
		} else if len(configs) == 0 {
			log.Printf("No configurations found. Skipping check.")
		} else {
			configChanged := false

			for i := range configs {
				log.Printf("📦 [%d/%d] Checking %s...", i+1, len(configs), configs[i].Repo)
				
				// 检查 Release
				if configs[i].MonitorRelease {
					log.Printf("  🔍 Checking releases for %s", configs[i].Repo)
					release, err := getLatestRelease(httpClient, configs[i].Repo)
					if err != nil {
						log.Printf("  ❌ Error fetching release for %s: %v", configs[i].Repo, err)
					} else if release != nil {
						if configs[i].LastReleaseID == nil || *configs[i].LastReleaseID != release.ID {
							// 首次不发送通知
							if configs[i].LastReleaseID != nil {
								log.Printf("  🆕 New release detected for %s: %s (ID: %d -> %d)", configs[i].Repo, release.TagName, *configs[i].LastReleaseID, release.ID)
								msg := Messages.NotifyRelease(configs[i].Repo, release.TagName, release.HTMLURL)
								targetID := configs[i].ChannelID
								if targetID == 0 {
									targetID = adminID
								}
								log.Printf("  📤 Sending release notification to %d", targetID)
								tg.sendMessage(targetID, msg, telegramParseModeMarkdown, true, "")
							} else {
								log.Printf("  ℹ️ Initial release recorded for %s: %s (ID: %d) - no notification sent", configs[i].Repo, release.TagName, release.ID)
							}
							latestID := release.ID
							configs[i].LastReleaseID = &latestID
							configChanged = true
						} else {
							log.Printf("  ✓ No new release for %s (current: %d)", configs[i].Repo, release.ID)
						}
					} else {
						log.Printf("  ℹ️ No releases found for %s", configs[i].Repo)
					}
				}

				// 检查 Commit
				if configs[i].MonitorCommit {
					branch := configs[i].Branch
					if branch == "" {
						log.Printf("  🔍 Fetching default branch for %s", configs[i].Repo)
						defaultBr, err := getRepoDefaultBranch(httpClient, configs[i].Repo)
						if err != nil {
							log.Printf("  ⚠️ Failed to get default branch for %s, using 'main': %v", configs[i].Repo, err)
							branch = "main"
						} else {
							branch = defaultBr
						}
						// 缓存到配置，下次无需再请求 API
						configs[i].Branch = branch
						configChanged = true
					}

					log.Printf("  🔍 Checking commits for %s:%s", configs[i].Repo, branch)
					commit, err := getLatestCommit(httpClient, configs[i].Repo, branch)
					if err != nil {
						log.Printf("  ❌ Error fetching commit for %s:%s: %v", configs[i].Repo, branch, err)
					} else if commit != nil {
						if configs[i].LastCommitSHA == nil || *configs[i].LastCommitSHA != commit.SHA {
							// 首次不发送通知
							if configs[i].LastCommitSHA != nil {
								oldSHA := "none"
								if configs[i].LastCommitSHA != nil {
									oldSHA = (*configs[i].LastCommitSHA)[:7]
								}
								log.Printf("  🆕 New commit detected for %s:%s: %.7s -> %.7s", configs[i].Repo, branch, oldSHA, commit.SHA)
								message := strings.TrimSpace(commit.Commit.Message)
								if message == "" {
									message = commit.SHA
								}

								// AI 翻译
								var translation string
								if translated, err := translateText(message); err != nil {
									log.Printf("  ⚠️ AI translation failed: %v", err)
								} else if translated != "" {
									translation = translated
								}

								repoName := configs[i].Repo
								if parts := strings.Split(configs[i].Repo, "/"); len(parts) == 2 {
									repoName = parts[1]
								}
								
								// 使用 Messages 构建消息
								msg := Messages.NotifyCommit(repoName, branch, message, translation, commit.HTMLURL)

								targetID := configs[i].ChannelID
								if targetID == 0 {
									targetID = adminID
								}
								log.Printf("  📤 Sending commit notification to %d", targetID)
								tg.sendMessage(targetID, msg, telegramParseModeMarkdown, true, "")
							} else {
								log.Printf("  ℹ️ Initial commit recorded for %s:%s: %.7s - no notification sent", configs[i].Repo, branch, commit.SHA)
							}
							latestSHA := commit.SHA
							configs[i].LastCommitSHA = &latestSHA
							configChanged = true
						} else {
							log.Printf("  ✓ No new commit for %s:%s (current: %.7s)", configs[i].Repo, branch, commit.SHA)
						}
					} else {
						log.Printf("  ℹ️ No commits found for %s:%s", configs[i].Repo, branch)
					}
				}

				time.Sleep(repoCheckDelay)
			}

			log.Printf("🎯 Check cycle complete for %d repositories", len(configs))
			if configChanged {
				log.Printf("🔄 Configuration changed, saving updates...")
				if err := saveConfigs(configs); err != nil {
					log.Printf("❌ Failed to save configs: %v", err)
				}
			} else {
				log.Printf("ℹ️ No configuration changes to save")
			}
		}

		log.Printf("Check finished. Waiting for %s.", checkInterval)
		time.Sleep(checkInterval)
	}
}
