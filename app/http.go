package clicktracker

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/taskqueue"
	"appengine/user"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/mjibson/appstats"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var templates = template.Must(template.New("clicks-index").Delims("[[", "]]").ParseGlob("templates/*.html"))
var clicksPerPage = 20

type TemplateData struct {
	LogoutURL  string
	User       string
	Error      string
	FormValues map[string]string
	Data       interface{}
}

func NewTemplateData(c appengine.Context) (TemplateData, error) {
	url, err := user.LogoutURL(c, "/")
	if err != nil {
		return TemplateData{}, err
	}

	return TemplateData{
		User:      user.Current(c).Email,
		LogoutURL: url,
	}, nil
}

func createClick(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		id, _ := strconv.ParseInt(v.Get(":id"), 10, 64)
		key := datastore.NewKey(c, "Tracker", "", id, nil)
		var tracker Tracker
		if err1 := datastore.Get(c, key, &tracker); err1 != nil {
			http.Error(w, err1.Error(), http.StatusInternalServerError)
			return err1
		}

		tracker.Count++

		sec, _ := strconv.ParseInt(r.FormValue("timestamp"), 10, 64)
		click := Click{
			RemoteAddr: r.FormValue("remote_addr"),
			UserAgent:  r.FormValue("user_agent"),
			Campaign:   r.FormValue("yoc_campaign"),
			Source:     r.FormValue("yoc_source"),
			Medium:     r.FormValue("yoc_medium"),
			Date:       time.Unix(sec, 0),
		}

		_, err1 := datastore.Put(c, datastore.NewIncompleteKey(c, "Click", key), &click)
		_, err1 = datastore.Put(c, key, &tracker)
		return err1
	}, nil)

	if err != nil {
		fmt.Fprintf(w, "error: %v", err)
		return
	}
}

func addTask(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	t := taskqueue.NewPOSTTask("/"+v.Get(":id")+"/clicks", map[string][]string{
		"yoc_campaign": {v.Get("yoc_campaign")},
		"yoc_source":   {v.Get("yoc_source")},
		"yoc_medium":   {v.Get("yoc_medium")},
		"remote_addr":  {r.RemoteAddr},
		"user_agent":   {r.UserAgent()},
		"timestamp":    {fmt.Sprintf("%v", time.Now().Unix())},
	})

	if _, err := taskqueue.Add(c, t, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func redirect(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	id, _ := strconv.ParseInt(v.Get(":id"), 10, 64)

	if item, err := memcache.Get(c, v.Get(":id")); err == nil {
		addTask(c, w, r)
		http.Redirect(w, r, string(item.Value), http.StatusFound)
	} else {
		var tracker Tracker
		if err := datastore.Get(c, datastore.NewKey(c, "Tracker", "", id, nil), &tracker); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		memcache.Add(c, &memcache.Item{
			Key:   v.Get(":id"),
			Value: []byte(tracker.URL),
		})

		addTask(c, w, r)
		http.Redirect(w, r, tracker.URL, http.StatusFound)
	}
}

func indexClicks(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	id, _ := strconv.ParseInt(values.Get(":id"), 10, 64)
	campaign := values.Get("campaign")
	medium := values.Get("medium")
	source := values.Get("source")

	key := datastore.NewKey(c, "Tracker", "", id, nil)
	var tracker Tracker
	if err := datastore.Get(c, key, &tracker); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tracker.IntID = id

	q := datastore.NewQuery("Click").Ancestor(key).Order("-Date")

	if campaign != "" {
		q = q.Filter("Campaign =", campaign)
	}

	if medium != "" {
		q = q.Filter("Medium =", medium)
	}

	if source != "" {
		q = q.Filter("Source =", source)
	}

	markers := make([]datastore.Cursor, 0, clicksPerPage)
	markerQuery := q.KeysOnly()
	i := 0
	for t := markerQuery.Run(c); ; {
		c.Infof("-- %v", i)
		if i%clicksPerPage == 0 {
			cursor, err := t.Cursor()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			markers = append(markers, cursor)
		}
		i++

		_, err := t.Next(nil)
		if err == datastore.Done {
			break
		}
	}

	c.Infof("\n\n\nMarkers %v", &markers)

	d, _ := NewTemplateData(c)

	p := ClickPage{
		Tracker:  tracker,
		Campaign: campaign,
		Medium:   medium,
		Source:   source,
		Markers:  markers,
	}

	if values.Get("page") != "" {
		p.Current, _ = strconv.Atoi(values.Get("page"))
		q = q.Start(markers[p.Current-1])
	} else {
		p.Current = 1
	}

	clicks := make([]Click, 0, clicksPerPage)
	q = q.Limit(clicksPerPage)
	_, err := q.GetAll(c, &clicks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.Clicks = clicks

	d.Data = p
	c.Infof("%v = %v", values.Get("page"), p.Current)

	if err := templates.ExecuteTemplate(w, "clicks-index.html", d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func trackerPage(c appengine.Context) (TrackerPage, error) {
	q := datastore.NewQuery("Tracker").Limit(10)

	trackers := make([]Tracker, 0, 10)

	keys, err := q.GetAll(c, &trackers)
	if err != nil {
		return TrackerPage{}, err
	}

	for k, _ := range keys {
		trackers[k].IntID = keys[k].IntID()
	}

	return TrackerPage{
		Trackers: trackers,
	}, nil
}

func indexTrackers(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	d, err := NewTemplateData(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d.Data, err = trackerPage(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "trackers-index.html", d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func createTracker(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse(r.FormValue("destination-url"))

	if err != nil || url.Scheme == "" || url.Host == "" {
		d, err := NewTemplateData(c)
		if err != nil {
			fmt.Fprintf(w, err.Error())
			return
		}

		d.Error = "Please type a valid URL, including its scheme (for example, \"http://\")"
		d.Data, err = trackerPage(c)
		d.FormValues = map[string]string{
			"DestinationURL": r.FormValue("destination-url"),
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := templates.ExecuteTemplate(w, "index-trackers.html", d); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	tracker := Tracker{
		URL:  url.String(),
		User: user.Current(c).Email,
	}

	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Tracker", nil), &tracker); err != nil {
		fmt.Fprintf(w, "error: %s", err)
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func blitz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%v", 42)
}

func init() {
	m := pat.New()

	m.Get("/mu-e1e6a634-72dd8fe0-c2b41298-fe1e2321", http.HandlerFunc(blitz))

	m.Post("/:id/clicks", appstats.NewHandler(createClick))
	m.Post("/trackers/:id/clicks", appstats.NewHandler(createClick))

	m.Get("/", appstats.NewHandler(indexTrackers))
	m.Get("/trackers", appstats.NewHandler(indexTrackers))
	m.Post("/trackers", appstats.NewHandler(createTracker))

	m.Get("/:id", appstats.NewHandler(redirect))
	m.Get("/trackers/:id", appstats.NewHandler(redirect))

	m.Get("/trackers/:id/clicks", appstats.NewHandler(indexClicks))
	m.Get("/trackers/:id/edit", appstats.NewHandler(indexClicks))

	http.Handle("/", m)
}
