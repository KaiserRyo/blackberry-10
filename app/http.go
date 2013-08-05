package blackberry

import (
	"appengine"
	"appengine/datastore"
	"appengine/taskqueue"
	"appengine/user"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/mjibson/appstats"
	"html/template"
	"net/http"
	"strconv"
	"time"
)

var templates = template.Must(template.New("signups-index").ParseGlob("templates/*.html"))
var signUpsPerPage = 20

type TemplateData struct {
	LogoutURL  string
	User       string
	Error      string
	FormValues map[string]string
	Data       interface{}
}

func newTemplateData(c appengine.Context) (TemplateData, error) {
	url, err := user.LogoutURL(c, "/")
	if err != nil {
		return TemplateData{}, err
	}

	return TemplateData{
		User:      user.Current(c).Email,
		LogoutURL: url,
	}, nil
}

func createSignUp(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	sec, _ := strconv.ParseInt(r.FormValue("timestamp"), 10, 64)
	signUp := SignUp{
		EmailAddr:  r.FormValue("email_addr"),
		RemoteAddr: r.FormValue("remote_addr"),
		UserAgent:  r.FormValue("user_agent"),
		Date:       time.Unix(sec, 0),
	}

	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "SignUp", nil), &signUp); err != nil {
		c.Infof("error: %v", err)
		return
	}
}

func createSignUpAsync(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	t := taskqueue.NewPOSTTask("/signups/task", map[string][]string{
		"email_addr":  {r.FormValue("email_addr")},
		"remote_addr": {r.RemoteAddr},
		"user_agent":  {r.UserAgent()},
		"timestamp":   {fmt.Sprintf("%v", time.Now().Unix())},
	})

	if _, err := taskqueue.Add(c, t, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", 302)
}

func indexSignUps(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	q := datastore.NewQuery("SignUp").Order("-Date")

	markers := make([]datastore.Cursor, 0, signUpsPerPage)
	markerQuery := q.KeysOnly()
	i := 0
	for t := markerQuery.Run(c); ; {
		c.Infof("-- %v", i)
		if i%signUpsPerPage == 0 {
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

	d, _ := newTemplateData(c)

	p := SignUpPage{
		Markers: markers,
	}

	if values.Get("page") != "" {
		p.Current, _ = strconv.Atoi(values.Get("page"))
		q = q.Start(markers[p.Current-1])
	} else {
		p.Current = 1
	}

	signups := make([]SignUp, 0, signUpsPerPage)
	q = q.Limit(signUpsPerPage)
	_, err := q.GetAll(c, &signups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.SignUps = signups

	d.Data = p
	c.Infof("%v = %v", values.Get("page"), p.Current)

	if err := templates.ExecuteTemplate(w, "signups-index.html", d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newSignUp(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "signups-new.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func init() {
	m := pat.New()
	m.Post("/signups", appstats.NewHandler(createSignUpAsync))
	m.Post("/signups/task", appstats.NewHandler(createSignUp))
	m.Get("/signups", appstats.NewHandler(indexSignUps))
	m.Get("/", appstats.NewHandler(newSignUp))
	http.Handle("/", m)
}
