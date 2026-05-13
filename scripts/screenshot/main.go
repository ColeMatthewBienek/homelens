//go:build ignore
// +build ignore

// scripts/screenshot/main.go renders an HTML file to PNG via chromedp for the
// README hero shot. Run with:
//
//   go run scripts/screenshot/main.go <input.html> <output.png>
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: screenshot <input.html> <output.png>")
		os.Exit(2)
	}
	abs, _ := filepath.Abs(os.Args[1])
	fileURL := "file:///" + filepath.ToSlash(abs)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(ctx,
		emulation.SetDeviceMetricsOverride(1400, 1750, 2.0, false),
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(1500*time.Millisecond),
		chromedp.CaptureScreenshot(&buf),
	); err != nil {
		fmt.Fprintln(os.Stderr, "screenshot:", err)
		os.Exit(3)
	}
	if err := os.WriteFile(os.Args[2], buf, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(3)
	}
	fmt.Println(os.Args[2])
}
