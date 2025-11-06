package main

import (
	"fmt"
	"time"
	"os"
	"log"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all" // register cookie store finders!
	"github.com/browserutils/kooky/browser/chrome"
)

type DSIDCookiePoller struct {
	cookiePath string
	domain     string
	cookieName string
	tmpFile    string
	lastDSID   string
	log        *log.Logger
}

func NewDSIDCookiePoller(config DsidCookiePollerConfig, tmpFile string) *DSIDCookiePoller {
	return &DSIDCookiePoller {
		cookiePath: config.CookiePath,
		domain: config.CookieHost,
		cookieName: config.CookieName,
	  tmpFile: tmpFile,
		log: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (poller *DSIDCookiePoller) openCookies() kooky.CookieSeq {
	return chrome.TraverseCookies(poller.cookiePath).OnlyCookies()
}

func (poller *DSIDCookiePoller) get() (string, error) {
	for cookie := range poller.openCookies() {
		if cookie.Domain == poller.domain && cookie.Name == poller.cookieName {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("[parent] DSID not found for domain %q", poller.domain)
}

func (p *DSIDCookiePoller) pollAndSave() {
	if dsid, err := p.get(); err == nil {
		if dsid != p.lastDSID {
			p.log.Printf("Found new DSID = %s, old dsid = %s, writing to %s\n", dsid, p.lastDSID, p.tmpFile)
			err = os.WriteFile(p.tmpFile, []byte(dsid), 0600)
			if err != nil {
				fmt.Errorf("Error writing DSID")
			}
			p.lastDSID = dsid
		}
		return
	}
}

func (p *DSIDCookiePoller) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.pollAndSave()
		}
	}
}


