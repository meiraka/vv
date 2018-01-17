package main

import (
	"bytes"
	"golang.org/x/text/language"
	"html/template"
)

var translatePrio = []language.Tag{
	language.AmericanEnglish,
	language.Japanese,
}

var translateData = map[language.Tag]map[string]string{
	language.AmericanEnglish: {},
	language.Japanese: {
		"lang":                        "ja",
		"Preferences":                 "環境設定",
		"Stats":                       "統計",
		"Information":                 "情報",
		"System":                      "システム",
		"ReloadApplication":           "再読み込み",
		"Appearance":                  "外観の設定",
		"ColorThreshold":              "色",
		"Animation":                   "アニメーション",
		"BackgroundImage":             "背景画像を表示",
		"BackgroundImageBlur":         "背景画像のブラーエフェクト",
		"CircledImage":                "カバーアートを丸く表示",
		"GridviewAlbum":               "アルバム一覧をグリッド表示",
		"AutoHideScrollbar":           "自動的にスクロールバーを非表示",
		"Library":                     "ライブラリ",
		"Rescan":                      "ライブラリを再読み込み",
		"Playback":                    "再生",
		"ListviewFollowsPlayback":     "次に再生する曲に自動で移動",
		"Volume":                      "音量",
		"ShowVolumeNob":               "音量バーを表示",
		"MaxVolume":                   "音量の最大値",
		"Devices":                     "デバイス",
		"Tracks":                      "トラック数:",
		"Artists":                     "アーティスト数:",
		"Albums":                      "アルバム数:",
		"TotalLength":                 "総トラック時間:",
		"TotalPlaytime":               "総再生時間:",
		"Uptime":                      "起動時間:",
		"LastLibraryUpdate":           "ライブラリの最終更新時刻:",
		"Websockets":                  "Websocket接続数",
		"ThisApplication":             "このアプリケーションについて",
		"Compiler":                    "コンパイラ",
		"BSD3ClauseLicense":           "修正BSDライセンス",
		"Renderer":                    "ブラウザ",
		"Credits":                     "クレジット",
		"MITLicense":                  "MIT ライセンス",
		"BSD2ClauseLicense":           "二条項BSDライセンス",
		"Help":                        "ヘルプ",
		"KeyboardShortcuts":           "キーボード・ショートカット",
		"PlayOrPauseSong":             "再生・一時停止",
		"GoToNextSong":                "次の曲",
		"GoToPreviousSong":            "前の曲",
		"MoveListViewCursor":          "カーソル移動",
		"ActivateListViewCursorItem":  "カーソル上の曲を再生/下のディレクトリへ移動",
		"ShowListParent":              "上のディレクトリへ移動",
		"ShowNowPlayingItem":          "現在再生中の曲へ移動",
		"ShowThisHelp":                "このヘルプを表示",
		"ariaLabelBackTo":             "%s 一覧に移動",
		"ariaLabelShowNowPlayingItem": "現在再生中の曲へ移動",
		"ariaLabelShowSettingsWindow": "設定ウィンドウを表示",
		"ariaLabelCloseThisWindow":    "ウィンドウを閉じる",
		"ariaLabelGoToPreviousSong":   "前の曲に戻る",
		"ariaLabelGoToNextSong":       "次の曲を再生",
		"ariaLabelPauseSong":          "一時停止",
		"ariaLabelPlaySong":           "再生",
		"ariaLabelTurnOnRepeat":       "リピート再生を有効にする",
		"ariaLabelTurnOffRepeat":      "リピート再生を無効にする",
		"ariaLabelTurnOnRandom":       "ランダム再生を有効にする",
		"ariaLabelTurnOffRandom":      "ランダム再生を無効にする",
		"titleFormatBackTo":           "%s に戻る",
		"titleShowNowPlayingItem":     "現在再生中の曲へ移動",
		"titleShowSettingsWindow":     "設定",
		"titleClose":                  "閉じる",
		"titlePrevious":               "前の曲",
		"titlePlayOrPause":            "再生・一時停止",
		"titleNext":                   "次の曲",
		"titleRepeat":                 "リピート",
		"titleRandom":                 "ランダム",
	},
}

func translate(html []byte, lang language.Tag) ([]byte, error) {
	t, err := template.New("webpage").Parse(string(html))
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, translateData[lang])
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
