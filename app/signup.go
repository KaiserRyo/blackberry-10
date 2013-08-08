package blackberry

import (
	"appengine/datastore"
	"fmt"
	"net/url"
	"time"
)

type SignUp struct {
	EmailAddr  string
	FirstName  string
	LastName   string
	RemoteAddr string
	UserAgent  string
	Date       time.Time
}

type SignUpPage struct {
	Current    int
	SignUps    []SignUp
	RemoteAddr string
	UserAgent  string
	Markers    []datastore.Cursor
}

func (p SignUpPage) SelfURL() string {
	v := p.URLValues()
	return fmt.Sprintf("/signups?%v", v.Encode())
}

func (p SignUpPage) DelFilterURL(param string) string {
	v := p.URLValues()
	v.Del(param)
	return fmt.Sprintf("/signups?%v", v.Encode())
}

func (p SignUpPage) AddFilterURL(key, value string) string {
	v := p.URLValues()
	v.Set(key, value)
	return fmt.Sprintf("/signups?%v", v.Encode())
}

func (p SignUpPage) NextURL() string {
	if p.Current >= len(p.Markers) {
		return ""
	}

	v := p.URLValues()
	v.Set("page", fmt.Sprintf("%v", p.Current+1))
	return fmt.Sprintf("/signups?%v", v.Encode())
}

func (p SignUpPage) PrevURL() string {
	if p.Current == 1 {
		return ""
	}

	v := p.URLValues()
	v.Set("page", fmt.Sprintf("%v", p.Current-1))
	return fmt.Sprintf("/signups?%v", v.Encode())
}

func (p SignUpPage) PageURL(page int) string {
	v := p.URLValues()
	v.Set("page", fmt.Sprintf("%v", page+1))
	return fmt.Sprintf("/signups?%v", v.Encode())
}

func (p SignUpPage) PageNumber(index int) int {
	return index + 1
}

func (p SignUpPage) IsCurrentPage(page int) bool {
	return p.Current == page+1
}

func (d SignUpPage) URLValues() url.Values {
	v := url.Values{}
	return v
}
