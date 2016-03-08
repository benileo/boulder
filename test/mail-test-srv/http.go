package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

// toFilter filters mails based on the To: field.
// The zero value matches all mails.
type toFilter struct {
	To string
}

func (f *toFilter) Match(m rcvdMail) bool {
	if f.To != "" && f.To != m.To {
		return false
	}
	return true
}

/*
/count - number of mails
/count?to=foo@bar.com - number of mails for foo@bar.com
/count?from=foo@bar.com - number of mails from foo@bar.com
/clear - clear the mail list
/mail/0 - first mail
/mail/1 - second mail
/mail/0?to=foo@bar.com - first mail for foo@bar.com
*/

func setupHTTP(serveMux *http.ServeMux) {
	serveMux.HandleFunc("/count", httpCount)
	serveMux.HandleFunc("/clear", httpClear)
	serveMux.HandleFunc("/mail/", httpGetMail)
}

func httpClear(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		allMailMutex.Lock()
		allReceivedMail = nil
		allMailMutex.Unlock()
		w.WriteHeader(200)
	} else {
		w.WriteHeader(405)
	}
}

func httpCount(w http.ResponseWriter, r *http.Request) {
	count := 0
	iterMail(extractFilter(r), func(m rcvdMail) bool {
		count++
		return false
	})
	fmt.Fprintf(w, "%d\n", count)
}

var rgxGetMail = regexp.MustCompile(`^/mail/(\d+)/?$`)

func httpGetMail(w http.ResponseWriter, r *http.Request) {
	match := rgxGetMail.FindStringSubmatch(r.URL.Path)
	if match == nil {
		w.WriteHeader(400)
		log.Println("mail-test-srv: bad request:", r.URL.Path)
		return
	}
	mailNum, err := strconv.Atoi(match[1])
	if err != nil {
		w.WriteHeader(400)
		log.Println("mail-test-srv: bad request:", r.URL.Path, "-", err)
		return
	}
	idx := 0
	found := iterMail(extractFilter(r), func(m rcvdMail) bool {
		if mailNum == idx {
			printMail(w, m)
			return true
		}
		idx++
		return false
	})
	if !found {
		w.WriteHeader(404)
	}
}

func extractFilter(r *http.Request) toFilter {
	values := r.URL.Query()
	to := values.Get("to")
	return toFilter{To: to}
}

func iterMail(f toFilter, cb func(rcvdMail) bool) bool {
	allMailMutex.Lock()
	defer allMailMutex.Unlock()
	for _, v := range allReceivedMail {
		if !f.Match(v) {
			continue
		}
		if cb(v) {
			return true
		}
	}
	return false
}

func printMail(w io.Writer, mail rcvdMail) {
	fmt.Fprintf(w, "FROM %s\n", mail.From)
	fmt.Fprintf(w, "TO %s\n", mail.To)
	fmt.Fprintf(w, "\n%s\n", mail.Mail)
}
