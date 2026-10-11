package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	INPUT_LIST        = "games.txt"
	OUTPUT_CSV        = "games.csv"
	LOCAL_CACHE       = "../cache"
	MAX_THREADS       = 5
	REQ_TIMEOUT       = 20 * time.Second
	MAX_SCRAPED_ITEMS = 500
	METACRITIC_URL    = "https://www.metacritic.com"
	GAMEPAGE_URL      = METACRITIC_URL + "/game/"
)

// UA strings
var UserAgents = []string{
	"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 10; LM-Q720) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 11; CPH2251) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 12; SM-A525F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 12; V2134) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; M2101K6G) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; SM-G991U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; SM-S908E) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.159 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.6167.85 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:120.0.1) Gecko/20100101 Firefox/120.0.1",
	"Mozilla/5.0 (iPad; CPU OS 16_7_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 17_0_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 15_7_9 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.5 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/118.0 Mobile/15E148 Safari/605.1.15",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0.1 Mobile/15E148 Safari/604.1",
}

// regex to replace bad characters in game names
var badCharsRegex = regexp.MustCompile(`[^a-z0-9]+`)

// rate limiter - token bucket
var rateLimiter chan struct{}

// GameScore -
type GameScore struct {
	Name        string
	CriticScore string
}

func extractScoreFromHtml(doc *goquery.Document, gameName string) string {
	var criticScore string

	// 1. Kontextová extrakce podle headeru
	doc.Find(`div[data-testid="product-score"]`).EachWithBreak(func(i int, s *goquery.Selection) bool {
		header := s.Find(`div[data-testid="global-score-header"]`).Text()
		if strings.EqualFold(strings.TrimSpace(header), "Metascore") {
			valSpan := s.Find(`span[data-testid="global-score-value"]`)
			if valSpan.Length() == 0 {
				valSpan = s.Find(`span[data-testid="global-score-tbd"]`)
			}
			criticScore = strings.ToLower(strings.TrimSpace(valSpan.Text()))
			return false
		}
		return true
	})

	score := "N/A"
	if strings.Contains(criticScore, ".") {
		log.Printf("⚠️ [%s] Detected dot in score (%s), forcing TBD", gameName, criticScore)
		criticScore = "tbd"
	}

	switch criticScore {
	case "":
		log.Printf("😡 [%s] N/A (score not found)", gameName)

	case "tbd":
		log.Printf("😐️ [%s] TBD", gameName)
		score = "TBD"
		htmlContent, err := doc.Html()
		if err != nil {
			log.Printf("❌ error getting content for [%s]: %v", gameName, err)
			return score
		}
		saveToCache(gameName, htmlContent)

	default:
		log.Printf("🥰 [%s] %s%%", gameName, criticScore)
		score = criticScore

		htmlContent, err := doc.Html()
		if err != nil {
			log.Printf("❌ error getting content for [%s]: %v", gameName, err)
			return score
		}
		saveToCache(gameName, htmlContent)
	}

	return score
}

// saveToCache - save HTML to a local cache
func saveToCache(gameName, content string) {
	if _, err := os.Stat(LOCAL_CACHE); os.IsNotExist(err) {
		err = os.MkdirAll(LOCAL_CACHE, 0755)
		if err != nil {
			log.Fatalf("💥 [%s] error creating folder [%s]: %v", gameName, LOCAL_CACHE, err)
		}
	}

	filePath := filepath.Join(LOCAL_CACHE, gameName+".html")
	if _, err := os.Stat(filePath); err == nil {
		return // file already exists
	} else if !os.IsNotExist(err) {
		log.Printf("💥 [%s] error checking file at [%s]: %v", gameName, filePath, err)
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Printf("💥 [%s] error saving: %v", gameName, err)
	}
}

// loadFromCache - try to load HTML from a local cache
func loadFromCache(gameName string) (*goquery.Document, error) {
	filePath := filepath.Join(LOCAL_CACHE, gameName+".html")
	localFileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	log.Printf("[%s] - reading cache", gameName)
	localDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(localFileContent)))
	if err != nil {
		log.Printf("😵‍💫 [%s] error creating document from cache: %v", gameName, err)
		return nil, err
	}
	return localDoc, nil
}

// searchAndScrape - find game page URL and then search for the score, extract data
func searchAndScrape(UA string, ctx context.Context, gameName string, results chan<- GameScore, wg *sync.WaitGroup) {
	defer wg.Done()

	// try cache first
	localDoc, err := loadFromCache(gameName)
	if err == nil {
		// try to find the score
		criticScore := extractScoreFromHtml(localDoc, gameName)
		results <- GameScore{Name: gameName, CriticScore: criticScore}
		return
	}

	// wait for a token from limiter
	select {
	case <-rateLimiter:
		// Máme token, pokračujeme
	case <-ctx.Done():
		// Kontext byl zrušen (Ctrl+C), končíme hned
		return
	}

	// search live
	query := strings.ToLower(gameName)
	query = badCharsRegex.ReplaceAllString(query, "-")
	query = strings.Trim(query, "-")
	searchUrl := fmt.Sprintf("%s%s/", GAMEPAGE_URL, query)
	log.Printf("🔎 [%s] at: %s", gameName, searchUrl)

	client := &http.Client{
		Timeout: REQ_TIMEOUT,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", searchUrl, nil)
	if err != nil {
		log.Printf("[%s] error in search: %v", gameName, err)
		results <- GameScore{Name: gameName, CriticScore: "Error"}
		return
	}

	req.Header.Set("User-Agent", UA)

	res, err := client.Do(req)
	if err != nil {
		log.Printf("[%s] error during search: %v", gameName, err)
		results <- GameScore{Name: gameName, CriticScore: "Error"}
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("[%s] search status code [%d] \"%s\" at %s", gameName, res.StatusCode, res.Status, searchUrl)
		results <- GameScore{Name: gameName, CriticScore: "N/A"}
		return
	}

	searchDoc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("[%s] error creating document: %v", gameName, err)
		results <- GameScore{Name: gameName, CriticScore: "Error"}
		return
	}

	// try to find the game score in the current page
	criticScoreX := extractScoreFromHtml(searchDoc, gameName)
	//if criticScoreX != "TBD" && criticScoreX != "N/A" {
	if criticScoreX != "N/A" {
		results <- GameScore{Name: gameName, CriticScore: criticScoreX}
		return
	}

	var gameURL string
	found := false
	searchDoc.Find("a.c-pageSiteSearch-results-item").Each(func(i int, s *goquery.Selection) {
		if found {
			return
		}

		url, exists := s.Attr("href")
		if !exists || !strings.Contains(url, "/game/") {
			return // ignore links not targeting games
		}

		// process title and game name
		title := s.Find("p.g-text-bold").Text()
		normalizedTitle := strings.TrimSpace(strings.ToLower(title))
		normalizedGameName := strings.TrimSpace(strings.ToLower(gameName))

		// strings to remove
		var toRemove = []string{
			"game of the year",
			"remake",
			"remaster",
			"enhanced edition",
			"definitive edition",
			"deluxe",
		}
		for _, r := range toRemove {
			normalizedGameName = strings.ReplaceAll(normalizedGameName, r, "")
			normalizedTitle = strings.ReplaceAll(normalizedTitle, r, "")
		}

		if strings.Contains(normalizedTitle, normalizedGameName) && strings.Contains(strings.ToLower(s.Text()), "pc") {
			url, exists := s.Attr("href")
			if exists {
				gameURL = METACRITIC_URL + url
				found = true
				return
			}
		}

		if strings.Contains(normalizedTitle, normalizedGameName) && strings.Contains(strings.ToLower(s.Text()), "xbox") {
			url, exists := s.Attr("href")
			if exists {
				gameURL = METACRITIC_URL + url
				found = true
				return
			}
		}

	})

	if gameURL == "" {
		log.Printf("❌ [%s] not found", gameName)
		results <- GameScore{Name: gameName, CriticScore: "N/A"}
		return
	}
	log.Printf("✅ [%s] found - %s", gameName, gameURL)

	// wait for a token from limiter
	<-rateLimiter

	req, err = http.NewRequestWithContext(ctx, "GET", gameURL, nil)
	if err != nil {
		log.Printf("[%s] error getting game page: %v", gameName, err)
		results <- GameScore{Name: gameName, CriticScore: "Error"}
		return
	}

	req.Header.Set("User-Agent", UA)

	res, err = client.Do(req)
	if err != nil {
		log.Printf("[%s] error during request: %v", gameName, err)
		results <- GameScore{Name: gameName, CriticScore: "Error"}
		return
	}
	defer res.Body.Close()

	var gameDoc *goquery.Document
	gameDoc, err = goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("[%s] error creating document: %v", gameName, err)
		results <- GameScore{Name: gameName, CriticScore: "Error"}
		return
	}

	criticScore := extractScoreFromHtml(gameDoc, gameName)
	results <- GameScore{Name: gameName, CriticScore: criticScore}
}

// loadCsv - load CSV file
func loadCsv(filename string) (map[string]bool, error) {
	scraped := make(map[string]bool)
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) { // no file means that we start fresh
			return scraped, nil
		}
		return nil, fmt.Errorf("error opening CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading from CSV: %w", err)
	}

	for _, record := range records {
		if len(record) > 0 {
			scraped[strings.TrimSpace(record[0])] = true
		}
	}
	return scraped, nil
}

// appendToCsv - CSV file, items separated by ';'
func appendToCsv(scores []GameScore, filename string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("💥 error opening [%s] for writing: %v", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = ';' // CSV separator
	for _, score := range scores {
		row := []string{score.Name, score.CriticScore}
		if err := writer.Write(row); err != nil {
			log.Fatalf("💥 error writing item to [%s]: %v", filename, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Fatalf("💥 error writing to [%s]: %v", filename, err)
	}
}

/* MAIN *** MAIN *** MAIN *** MAIN *** MAIN *** MAIN *** MAIN *** MAIN *** MAIN */

func main() {
	log.SetFlags(0)

	// set random UA
	UA := UserAgents[rand.Intn(len(UserAgents))]
	log.Printf("User Agent: %s", UA)

	// initialize token bucket
	rate := time.Duration(rand.Intn(20)+10) * time.Second
	rateLimiter = make(chan struct{}, 1)
	go func() {
		for {
			rateLimiter <- struct{}{}
			time.Sleep(rate)
		}
	}()

	// load main data
	file, err := os.Open(INPUT_LIST)
	if err != nil {
		log.Fatalf("💥 error opening [%s]: %v", INPUT_LIST, err)
	}
	defer file.Close()

	var allGames []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		gameName := strings.TrimSpace(scanner.Text())
		if gameName != "" {
			allGames = append(allGames, gameName)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("💥 error reading [%s]: %v", INPUT_LIST, err)
	}
	if len(allGames) == 0 {
		log.Printf("File [%s] is empty, nothing to scrape.", INPUT_LIST)
		return
	}

	// shuffle games to scrape randomly
	rand.Shuffle(len(allGames), func(i, j int) {
		allGames[i], allGames[j] = allGames[j], allGames[i]
	})

	// load existing CSV
	scrapedGames, err := loadCsv(OUTPUT_CSV)
	if err != nil {
		log.Fatalf("💥 error loading [%s]: %v", OUTPUT_CSV, err)
	}

	var gamesToScrape []string
	for _, game := range allGames {
		if strings.ContainsAny(game, "[]%") {
			continue
		}
		if _, exists := scrapedGames[game]; exists {
		} else {
			gamesToScrape = append(gamesToScrape, game)
		}
	}

	// max. scraping limits
	if len(gamesToScrape) > MAX_SCRAPED_ITEMS {
		gamesToScrape = gamesToScrape[:MAX_SCRAPED_ITEMS]
	}

	if len(gamesToScrape) == 0 {
		log.Println("\n🍀 Everything has already been scraped.")
		return
	}

	results := make(chan GameScore, len(gamesToScrape))
	var wg sync.WaitGroup
	var newScrapedScores []GameScore

	ctx, cancel := context.WithCancel(context.Background())

	// signals handling
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		log.Println("\n\nInterrupt received ...")
		cancel()
	}()

	// limit concurrency
	concurrencyLimit := make(chan struct{}, MAX_THREADS)

	// start scraping
	for _, game := range gamesToScrape {
		wg.Add(1)
		concurrencyLimit <- struct{}{}
		go func(game string) {
			defer func() {
				<-concurrencyLimit
			}()
			searchAndScrape(UA, ctx, game, results, &wg)
		}(game)
	}

	// wait for scrapers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// collect results
	for score := range results {
		newScrapedScores = append(newScrapedScores, score)
	}

	// sort and save
	sort.Slice(newScrapedScores, func(i, j int) bool {
		return newScrapedScores[i].Name < newScrapedScores[j].Name
	})
	appendToCsv(newScrapedScores, OUTPUT_CSV)
	log.Printf("🍀 Scraping finished.")
}
