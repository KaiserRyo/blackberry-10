package blackberry

import (
	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"appengine/taskqueue"
	"appengine/user"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/mjibson/appstats"
	"html/template"
	"net/http"
	"net/url"
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
		FirstName:  r.FormValue("first_name"),
		LastName:   r.FormValue("last_name"),
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
		"first_name":  {r.FormValue("first_name")},
		"last_name":   {r.FormValue("last_name")},
		"remote_addr": {r.RemoteAddr},
		"user_agent":  {r.UserAgent()},
		"timestamp":   {fmt.Sprintf("%v", time.Now().Unix())},
	})

	if _, err := taskqueue.Add(c, t, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	v := url.Values{}
	v.Set("message", "Vielen Dank für Ihre Anfrage. Wir werden uns in Kürze mit Ihnen in Verbindung setzen.")

	http.Redirect(w, r, "/?"+v.Encode(), 302)
}

func newSignUp(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	message := v.Get("message")
	if err := templates.ExecuteTemplate(w, "signups-new.html", message); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sendEmail(c appengine.Context, w http.ResponseWriter, r *http.Request) {
	buffer := new(bytes.Buffer)
	writer := csv.NewWriter(buffer)

	signups := make([]SignUp, 0, signUpsPerPage)
	q := datastore.NewQuery("SignUp").Order("-Date")
	_, err := q.GetAll(c, &signups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lines := make([][]string, 0, 10)
	for _, v := range signups {
		lines = append(lines, []string{v.FirstName, v.LastName, v.EmailAddr, v.Date.String(), v.RemoteAddr, v.UserAgent})
	}

	if err := writer.WriteAll(lines); err != nil {
		c.Errorf("Couldn't write CSV: %v", err)
	}

	att := mail.Attachment{
		Name: "all.csv",
		Data: buffer.Bytes(),
	}

	msg := &mail.Message{
		Sender:      "Quico Moya <quico.moya@yoc.com>",
		To:          []string{"de.blackberry@erseurope.com"},
		Bcc:         []string{"yasemin.kaya@yoc.com"},
		Subject:     "Blackberry 10 Campaign Report",
		Body:        fmt.Sprintf(""),
		Attachments: []mail.Attachment{att},
	}
	if err := mail.Send(c, msg); err != nil {
		c.Errorf("Couldn't send email: %v", err)
	}
}

func init() {
	m := pat.New()
	m.Post("/signups", appstats.NewHandler(createSignUpAsync))
	m.Post("/signups/task", appstats.NewHandler(createSignUp))
	m.Get("/sendemail", appstats.NewHandler(sendEmail))
	m.Get("/", appstats.NewHandler(newSignUp))
	http.Handle("/", m)
}
