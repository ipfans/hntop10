package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/systray"
	_ "github.com/glebarez/go-sqlite"
	"github.com/go-co-op/gocron"
	"github.com/go-resty/resty/v2"
	"github.com/martinlindhe/notify"
)

var (
	scheduler *gocron.Scheduler
	db        *sql.DB
	//go:embed Icon.png
	icon []byte
	//go:embed new.png
	newIcon []byte
)

func main() {
	home, _ := os.UserHomeDir()
	dbFile := filepath.Join(home, ".cache", "hackernews.db")
	var firstInit bool
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		firstInit = true
	}
	var err error
	db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		panic(err)
	}
	if firstInit {
		db.Exec("CREATE TABLE IF NOT EXISTS hn (id INTEGER PRIMARY KEY, title TEXT, url TEXT, raw_url TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)")
	}
	scheduler = gocron.NewScheduler(time.Local)
	systray.Run(onReady, onExit)
}

func onReady() {
	scheduler.Every(1).Hour().Do(refresh)
	scheduler.StartAsync()
	systray.SetIcon(icon)
	systray.SetTooltip("Hacker news Top10")
	systray.AddMenuItem("Loading...", "Loading...").Disable()
	exit()
}

func exit() {
	systray.AddSeparator()
	mQuitOrig := systray.AddMenuItem("Quit", "Quit")
	go func() {
		<-mQuitOrig.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	scheduler.Stop()
}

func openURL(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	exec.Command(cmd, args...).Start()
}

func refresh() {
	client := resty.New().SetBaseURL(baseURL)
	items, _ := topNHN(context.TODO(), client, topN)
	systray.ResetMenu()

	var count int

	if len(items) > 0 {
		for i := range items {
			var isNew bool
			url := items[i].RawURL
			if url == "" {
				url = items[i].URL
			}
			var id int64
			err := db.QueryRow("SELECT id FROM hn WHERE id = ?", items[i].ID).Scan(&id)
			if err != nil {
				count++
				isNew = true
			}
			db.Exec("INSERT OR IGNORE INTO hn (id, title, url, raw_url) VALUES (?, ?, ?, ?)", items[i].ID, items[i].Title, items[i].URL, items[i].RawURL)
			mItem := systray.AddMenuItem(items[i].Title, items[i].Title)
			if isNew {
				mItem.SetIcon(newIcon)
			}
			go func(url string) {
				<-mItem.ClickedCh
				openURL(url)
			}(url)
		}
		if count > 0 {
			notify.Notify("hntop10", "Hacker news Top10", fmt.Sprintf("Found %d new items", count), "")
		}
	} else {
		item := systray.AddMenuItem("No news found", "No news found")
		item.Disable()
	}
	exit()
}
