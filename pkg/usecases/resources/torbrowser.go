package resources

import (
	"net/url"
	"strings"
)

type TorBrowserLinks struct {
	DownloadLinks []url.URL
}

func (l *TorBrowserLinks) String() string {
	var s []string
	for _, link := range l.DownloadLinks {
		s = append(s, link.String())
	}
	return strings.Join(s, "\n")
}

func (l *TorBrowserLinks) Name() string {
	return "tblinks"
}

func (l *TorBrowserLinks) IsDepleted() bool {
	// Our links are public and therefore never depleted.
	return false
}

func (l *TorBrowserLinks) IsPublic() bool {
	return true
}
