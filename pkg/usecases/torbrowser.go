package usecases

import (
	"net/url"
	"strings"
)

type TorBrowserLinks struct {
	DownloadLinks []url.URL
}

func (l *TorBrowserLinks) String() string {
	return strings.Join(l.DownloadLinks, "\n")
}

func (l *TorBrowserLinks) IsDepleted() bool {
	// Our links are public and therefore never depleted.
	return false
}

func (l *TorBrowserLinks) IsPublic() bool {
	return true
}
