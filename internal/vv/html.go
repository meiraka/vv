package vv

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/meiraka/vv/internal/vv/assets"
)

// HTMLConfig is options for HTMLHandler.
type HTMLConfig struct {
	Local     bool      // use local asset file(assets/app.html)
	LocalDate time.Time // Last-Modified value for Local option
	LocalDir  string    // path to asset files directory
	Tree      Tree      // playlist view definition.
	TreeOrder []string  // order of playlist tree.
}

// NewHTMLHander creates http.Handler for app root html.
func NewHTMLHander(config *HTMLConfig) (http.Handler, error) {
	c := new(HTMLConfig)
	if config != nil {
		*c = *config
	}
	if c.Tree == nil && c.TreeOrder == nil {
		c.Tree = DefaultTree
		c.TreeOrder = DefaultTreeOrder
	}
	if c.Tree == nil && c.TreeOrder != nil {
		return nil, errors.New("invalid config: no tree")
	}
	if c.Tree != nil && c.TreeOrder == nil {
		return nil, errors.New("invalid config: no tree order")
	}
	if c.LocalDir == "" {
		c.LocalDir = "assets"
	}
	extra := map[string]string{
		"AssetsAppCSSHash": string(assets.AppCSSHash),
		"AssetsAppJSHash":  string(assets.AppJSHash),
	}
	jsonTree, err := json.Marshal(c.Tree)
	if err != nil {
		return nil, fmt.Errorf("tree: %v", err)
	}
	extra["TREE"] = string(jsonTree)
	jsonTreeOrder, err := json.Marshal(c.TreeOrder)
	if err != nil {
		return nil, fmt.Errorf("tree order: %v", err)
	}
	extra["TREE_ORDER"] = string(jsonTreeOrder)
	if c.Local {
		return i18nLocalHandler(filepath.Join(c.LocalDir, "app.html"), c.LocalDate, extra)
	}
	return i18nHandler(filepath.Join("assets", "app.html"), assets.AppHTML, extra)
}
