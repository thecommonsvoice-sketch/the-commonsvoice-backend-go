package services

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// NotifyArticlePublished performs fire-and-forget SEO notifications when an article is published.
// It pings Google/Bing sitemaps and triggers ISR revalidation on the frontend.
// All operations are non-blocking and failures are logged but never propagated.
func NotifyArticlePublished(frontendURL, articleSlug string) {
	go func() {
		siteURL := frontendURL
		sitemapURL := siteURL + "/articles-sitemap.xml"

		var wg sync.WaitGroup
		client := &http.Client{Timeout: 10 * time.Second}

		results := make([]string, 3)

		// 1. Ping Google
		wg.Add(1)
		go func() {
			defer wg.Done()
			pingURL := "https://www.google.com/ping?sitemap=" + url.QueryEscape(sitemapURL)
			resp, err := client.Get(pingURL)
			if err != nil {
				slog.Error("[SEO] Google ping failed (non-critical)", "error", err.Error())
				results[0] = "Google: ❌"
				return
			}
			resp.Body.Close()
			slog.Info("[SEO] Google sitemap ping", "status", resp.Status)
			results[0] = "Google: ✅"
		}()

		// 2. Ping Bing
		wg.Add(1)
		go func() {
			defer wg.Done()
			pingURL := "https://www.bing.com/ping?sitemap=" + url.QueryEscape(sitemapURL)
			resp, err := client.Get(pingURL)
			if err != nil {
				slog.Error("[SEO] Bing ping failed (non-critical)", "error", err.Error())
				results[1] = "Bing: ❌"
				return
			}
			resp.Body.Close()
			slog.Info("[SEO] Bing sitemap ping", "status", resp.Status)
			results[1] = "Bing: ✅"
		}()

		// 3. ISR Revalidation
		wg.Add(1)
		go func() {
			defer wg.Done()
			revalidatePaths := []string{
				"/articles-sitemap.xml",
				"/articles",
				"/",
			}
			if articleSlug != "" {
				revalidatePaths = append(revalidatePaths, "/articles/"+articleSlug)
			}

			allOK := true
			for _, path := range revalidatePaths {
				revalURL := fmt.Sprintf("%s/api/revalidate?path=%s", siteURL, url.QueryEscape(path))
				resp, err := client.Get(revalURL)
				if err != nil {
					slog.Error("[SEO] Frontend revalidation failed", "path", path, "error", err.Error())
					allOK = false
					continue
				}
				resp.Body.Close()
				slog.Info("[SEO] Revalidation", "path", path, "status", resp.Status)
			}
			if allOK {
				results[2] = "ISR Revalidation: ✅"
			} else {
				results[2] = "ISR Revalidation: ❌"
			}
		}()

		wg.Wait()
		slog.Info("[SEO] Publish notifications", "summary", fmt.Sprintf("%s | %s | %s", results[0], results[1], results[2]))
	}()
}
