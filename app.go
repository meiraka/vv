package main

import (
	"context"
	"net/http"

	"github.com/meiraka/vv/internal/mpd"
)

// NewHTTPHandler creates MPD http handler
func (h HTTPHandlerConfig) NewHTTPHandler(ctx context.Context, cl *mpd.Client, w *mpd.Watcher) (http.Handler, error) {
	api, err := h.NewAPIHandler(ctx, cl, w)
	if err != nil {
		return nil, err
	}
	m := http.NewServeMux()
	m.Handle("/", h.i18nAssetsHandler("assets/app.html", AssetsAppHTML, AssetsAppHTMLHash))
	m.Handle("/assets/app.css", h.assetsHandler("assets/app.css", AssetsAppCSS, AssetsAppCSSHash))
	m.Handle("/assets/app.png", h.assetsHandler("assets/app.png", AssetsAppPNG, AssetsAppPNGHash))
	m.Handle("/assets/manifest.json", h.assetsHandler("assets/manifest.json", AssetsManifestJSON, AssetsManifestJSONHash))
	m.Handle("/assets/app-black.png", h.assetsHandler("assets/app-black.png", AssetsAppBlackPNG, AssetsAppBlackPNGHash))
	m.Handle("/assets/w.png", h.assetsHandler("assets/w.png", AssetsWPNG, AssetsWPNGHash))
	m.Handle("/assets/app.js", h.assetsHandler("assets/appv2.js", AssetsAppv2JS, AssetsAppv2JSHash))
	m.Handle("/assets/nocover.svg", h.assetsHandler("assets/nocover.svg", AssetsNocoverSVG, AssetsNocoverSVGHash))
	m.Handle("/api/", api)
	return m, nil
}
