package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/ory/common/env"
	"github.com/ory/hydra/sdk/go/hydra"
	"github.com/ory/hydra/sdk/go/hydra/swagger"
	"github.com/pkg/errors"
	"github.com/urfave/negroni"
)

// To save user auth
var store = sessions.NewCookieStore([]byte("something-very-secret-keep-it-safe"))

// The session is a unique session identifier
const sessionName = "authentication"
const consentName = "consentsession"

var client hydra.SDK

func main() {
	var err error

	// Initialize the hydra SDK.
	client, err = hydra.NewSDK(&hydra.Configuration{
		ClientID:     env.Getenv("HYDRA_CLIENT_ID", "consent_app"),
		ClientSecret: env.Getenv("HYDRA_CLIENT_SECRET", "thisisthesecret"),
		EndpointURL:  env.Getenv("HYDRA_CLUSTER_URL", "http://hydra:4444"),
		Scopes:       []string{"hydra.* openid offline email preferredLanguage hydra"},
	})
	if err != nil {
		log.Fatalf("Unable to connect to the Hydra SDK because %s", err)
	}

	// Set up a router and some routes
	r := mux.NewRouter()
	r.HandleFunc("/consent", handleConsent)
	r.HandleFunc("/login", handleLogin)
	r.HandleFunc("/logout", handleLogout)

	// Set up a request logger, useful for debugging
	n := negroni.New()
	// n.Use(negronilogrus.NewMiddleware())
	n.UseHandler(r)

	// Start http server
	log.Println("Listening on :" + env.Getenv("PORT", "3000"))
	http.ListenAndServe(":"+env.Getenv("PORT", "3000"), n)
}

// Pressing "click here" the Authorize Code flow is performed and the user is redirected to hydra.
// Next, hydra validates the consent request (it's not valid yet) and redirects us to the consent endpoint which we set with `CONSENT_URL=http://localhost:4445/consent`.
func handleConsent(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)

	// Get the consent request id from the query.
	consentRequestID := r.URL.Query().Get("consent")
	if consentRequestID == "" {
		http.Error(w, errors.New("Consent endpoint was called without a consent request id").Error(), http.StatusBadRequest)
		return
	}

	// Fetch consent information
	consentRequest, response, err := client.GetOAuth2ConsentRequest(consentRequestID)
	if err != nil {
		http.Error(w, errors.Wrap(err, "The consent request endpoint does not respond").Error(), http.StatusBadRequest)
		return
	} else if response.StatusCode != http.StatusOK {
		http.Error(w, errors.Wrapf(err, "Consent request endpoint gave status code %d but expected %d", response.StatusCode, http.StatusOK).Error(), http.StatusBadRequest)
		return
	}

	// This little helper checks if our user is already authenticated. If not, we will redirect him to the login endpoint.
	user := authenticated(r)
	if user == "" {
		http.Redirect(w, r, "/login?consent="+consentRequestID, http.StatusFound)
		return
	}

	// Apparently, the user is logged in. Now we check if we received POST request, or a GET request.
	if r.Method == "POST" {
		// Ok, apparently the user gave his consent!

		// Parse the HTTP form
		if err := r.ParseForm(); err != nil {
			http.Error(w, errors.Wrap(err, "Could not parse form").Error(), http.StatusBadRequest)
			return
		}

		// Let's check which scopes the user granted.
		var grantedScopes = []string{}
		var extraScopes = map[string]interface{}{}
		for key := range r.PostForm {
			// And add each scope to the list of granted scopes.
			grantedScopes = append(grantedScopes, key)

			if key != "offline" && key != "openid" {
				extraScopes[key] = session.Values[key]
			}
		}

		// Ok, now we accept the consent request.
		response, err := client.AcceptOAuth2ConsentRequest(consentRequestID, swagger.ConsentRequestAcceptance{
			// The subject is a string, usually the user id.
			Subject: user,

			// The scopes our user granted.
			GrantScopes: grantedScopes,

			// Data that will be available on the token introspection and warden endpoints.
			AccessTokenExtra: map[string]interface{}{"foo": "bar"},

			// If we issue an ID token, we can set extra data for that id token here.
			IdTokenExtra: extraScopes,
		})
		if err != nil {
			http.Error(w, errors.Wrap(err, "The accept consent request endpoint encountered a network error").Error(), http.StatusInternalServerError)
			return
		} else if response.StatusCode != http.StatusNoContent {
			http.Error(w, errors.Wrapf(err, "Accept consent request endpoint gave status code %d but expected %d", response.StatusCode, http.StatusNoContent).Error(), http.StatusInternalServerError)
			return
		}

		// Redirect the user back to hydra, and append the consent response! If the user denies request you can
		// either handle the error in the authentication endpoint, or redirect the user back to the original application
		// with:
		//
		//   response, err := client.RejectOAuth2ConsentRequest(consentRequestID, payload)
		http.Redirect(w, r, consentRequest.RedirectUrl, http.StatusFound)
		return
	}

	// We received a get request, so let's show the html site where the user gives his consent.
	renderTemplate(w, "consent.html", struct {
		*swagger.OAuth2ConsentRequest
		ConsentRequestID string
	}{OAuth2ConsentRequest: consentRequest, ConsentRequestID: consentRequestID})
}

// The user hits this endpoint if he is not authenticated.
func handleLogin(w http.ResponseWriter, r *http.Request) {
	consentRequestID := r.URL.Query().Get("consent")

	type Err struct {
		Message error
	}
	var errText = new(Err)

	// Is it a POST request?
	if r.Method == "POST" {
		// Parse the form
		if err := r.ParseForm(); err != nil {
			http.Error(w, errors.Wrap(err, "Could not parse form").Error(), http.StatusBadRequest)
			return
		}

		username := r.Form.Get("username")
		password := r.Form.Get("password")

		resource, err := GetUserByCredential(username, password)
		if err != nil {
			errText.Message = err
			http.Redirect(w, r, "/login?consent="+consentRequestID, http.StatusAccepted)
			renderTemplate(w, "login.html", errText)
			return
		}

		// Let's create a session where we store the user id. We can ignore errors from the session store
		// as it will always return a session!
		session, _ := store.Get(r, sessionName)
		session.Options = &sessions.Options{
			Path:   "/",
			MaxAge: 300,
		}
		session.Values["user"] = resource.UserName
		session.Values["preferredLanguage"] = resource.Language
		session.Values["email"] = resource.Emails[0].Value

		// Store the session in the cookie
		if err := store.Save(r, w, session); err != nil {
			http.Error(w, errors.Wrap(err, "Could not persist cookie").Error(), http.StatusBadRequest)
			return
		}

		// Redirect the user back to the consent endpoint. In a normal app, you would probably
		// add some logic here that is triggered when the user actually performs authentication and is not
		// part of the consent flow.
		http.Redirect(w, r, "/consent?consent="+consentRequestID, http.StatusFound)
		return
	}

	// It's a get request, so let's render the template
	renderTemplate(w, "login.html", struct {
		*Err
		ConsentRequestID string
	}{Err: errText, ConsentRequestID: consentRequestID})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	consent, _ := store.Get(r, consentName)
	session.Options = &sessions.Options{
		MaxAge: -1,
	}
	consent.Options = &sessions.Options{
		MaxAge: -1,
	}
	session.Save(r, w)
	consent.Save(r, w)
	http.Redirect(w, r, "http://localhost:4445", http.StatusFound)
}

// authenticated checks if our cookie store has a user stored and returns the user's name, or an empty string if he is not authenticated.
func authenticated(r *http.Request) (user string) {
	session, _ := store.Get(r, sessionName)
	if u, ok := session.Values["user"]; !ok {
		return ""
	} else if user, ok = u.(string); !ok {
		return ""
	}
	return user
}

// renderTemplate is a convenience helper for rendering templates.
func renderTemplate(w http.ResponseWriter, id string, d interface{}) bool {
	if t, err := template.New(id).ParseFiles("./templates/" + id); err != nil {
		http.Error(w, errors.Wrap(err, "Could not render template").Error(), http.StatusInternalServerError)
		return false
	} else if err := t.Execute(w, d); err != nil {
		http.Error(w, errors.Wrap(err, "Could not render template").Error(), http.StatusInternalServerError)
		return false
	}
	return true
}
