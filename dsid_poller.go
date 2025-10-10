package main

import (
	"fmt"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all" // register cookie store finders!
	"github.com/browserutils/kooky/browser/chrome"
)

type DSIDPoller struct {
	cookiePath string
	domain     string
	cookieName string
}

func NewDSIDPoller(config DsidPollerConfig) *DSIDPoller {
	return &DSIDPoller{cookiePath: config.CookiePath, domain: config.CookieHost, cookieName: config.CookieName}
}

func (poller *DSIDPoller) openCookies() kooky.CookieSeq {
	return chrome.TraverseCookies(poller.cookiePath).OnlyCookies()
}

func (poller *DSIDPoller) get() (string, error) {
	for cookie := range poller.openCookies() {
		if cookie.Domain == poller.domain && cookie.Name == poller.cookieName {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("[parent] DSID not found for domain %q", poller.domain)
}
