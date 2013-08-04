package clicktracker

import (
	"appengine/datastore"
	"fmt"
	"net/url"
	"time"
)

type Click struct {
	RemoteAddr string
	UserAgent  string
	Campaign   string
	Source     string
	Medium     string
	Date       time.Time
}

type ClickPage struct {
	Current    int
	Clicks     []Click
	Tracker    Tracker
	RemoteAddr string
	UserAgent  string
	Campaign   string
	Source     string
	Medium     string
	Markers    []datastore.Cursor
}

func (d ClickPage) URLValues() url.Values {
	v := url.Values{}
	if d.Campaign != "" {
		v.Set("campaign", d.Campaign)
	}

	if d.Medium != "" {
		v.Set("medium", d.Medium)
	}

	if d.Source != "" {
		v.Set("source", d.Source)
	}

	return v
}

func (p ClickPage) SelfURL() string {
	v := p.URLValues()
	return fmt.Sprintf("/trackers/%v/clicks?%v", p.Tracker.IntID, v.Encode())
}

func (p ClickPage) DelFilterURL(param string) string {
	v := p.URLValues()
	v.Del(param)
	return fmt.Sprintf("/trackers/%v/clicks?%v", p.Tracker.IntID, v.Encode())
}

func (p ClickPage) AddFilterURL(key, value string) string {
	v := p.URLValues()
	v.Set(key, value)
	return fmt.Sprintf("/trackers/%v/clicks?%v", p.Tracker.IntID, v.Encode())
}

func (p ClickPage) NextURL() string {
	if p.Current >= len(p.Markers) {
		return ""
	}

	v := p.URLValues()
	v.Set("page", fmt.Sprintf("%v", p.Current+1))
	return fmt.Sprintf("/trackers/%v/clicks?%v", p.Tracker.IntID, v.Encode())
}

func (p ClickPage) PrevURL() string {
	if p.Current == 1 {
		return ""
	}

	v := p.URLValues()
	v.Set("page", fmt.Sprintf("%v", p.Current-1))
	return fmt.Sprintf("/trackers/%v/clicks?%v", p.Tracker.IntID, v.Encode())
}

func (p ClickPage) PageURL(page int) string {
	v := p.URLValues()
	v.Set("page", fmt.Sprintf("%v", page+1))
	return fmt.Sprintf("/trackers/%v/clicks?%v", p.Tracker.IntID, v.Encode())
}

func (p ClickPage) PageNumber(index int) int {
	return index + 1
}

func (p ClickPage) IsCurrentPage(page int) bool {
	return p.Current == page+1
}

func (p ClickPage) ShowURLBuilder() bool {
	return true
}
