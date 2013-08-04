package clicktracker

import (
	"fmt"
	"net/url"
)

type Tracker struct {
	IntID int64 `datastore:"-"`
	URL   string
	Count int
	User  string
}

type TrackerPage struct {
	Number   int
	Trackers []Tracker
}

func (t Tracker) SelfURL(campaign, medium, source string) string {
	v := url.Values{}

	if campaign != "" {
		v.Set("campaign", campaign)
	}

	if medium != "" {
		v.Set("medium", medium)
	}

	if source != "" {
		v.Set("source", source)
	}

	return fmt.Sprintf("/trackers/%v?", t.IntID) + v.Encode()
}

func (t Tracker) EditURL() string {
	return fmt.Sprintf("/trackers/%v/edit", t.IntID)
}

func (t Tracker) ClicksURL() string {
	return fmt.Sprintf("/trackers/%v/clicks", t.IntID)
}

func (t Tracker) QRCodeURL(campaign, medium, source string) string {
	v := url.Values{}
	v.Set("chs", "300x300")
	v.Set("cht", "qr")
	v.Set("choe", "UTF-8")
	v.Set("chl", "http://yoc-ly.appspot.com"+t.SelfURL(campaign, medium, source))
	return fmt.Sprintf("//chart.apis.google.com/chart?%v", v.Encode())
}

func (p TrackerPage) ShowURLBuilder() bool {
	return false
}
