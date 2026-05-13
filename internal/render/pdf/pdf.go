// Package pdf renders an HTML file to PDF via headless Chrome (chromedp).
// Requires Chrome / Chromium / Edge installed on the host.
package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// FromHTMLFile takes a path to an HTML file and writes a PDF to out.
// Stripping animations / hover effects happens on the print stylesheet
// in each theme via @media print rules.
func FromHTMLFile(htmlPath, out string) error {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	fileURL := "file:///" + filepath.ToSlash(abs)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var pdfBytes []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(800*time.Millisecond), // give Leaflet a beat to render if present
		chromedp.ActionFunc(func(ctx context.Context) error {
			b, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithMarginTop(0.4).WithMarginBottom(0.4).
				WithMarginLeft(0.4).WithMarginRight(0.4).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBytes = b
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("chromedp render: %w", err)
	}
	return os.WriteFile(out, pdfBytes, 0644)
}
