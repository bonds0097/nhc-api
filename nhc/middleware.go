package nhc

import (
	"errors"
	"net/http"
	"strings"

	"gopkg.in/mgo.v2"

	"github.com/codegangsta/negroni"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
)

func HeaderMiddleware() negroni.Handler {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if ENV == "prod" || ENV == "test" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next(w, r)
	})
}

func JWTMiddleware() negroni.Handler {
	ctx := Logger.WithField("method", "JWTMiddleware")
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if h := r.Header.Get("Authorization"); h != "" {
			token, err := jwt.ParseFromRequest(r, func(token *jwt.Token) (interface{}, error) {
				return VerifyKey, nil
			})

			switch err.(type) {
			case nil:
				if !token.Valid {
					NotAllowed(w, r)
					return
				}
				context.Set(r, "token", token)
				next(w, r)
			case *jwt.ValidationError:
				vErr := err.(*jwt.ValidationError)
				switch vErr.Errors {
				case jwt.ValidationErrorExpired:
					BR(w, r, errors.New("Token Expired"), http.StatusUnauthorized)
					return
				default:
					ISR(w, r, errors.New(vErr.Error()))
					ctx.Println(vErr.Error())
					return
				}
			default:
				ISR(w, r, err)
				return
			}
		} else {
			next(w, r)
		}
	})
}
func DBMiddleware(session *mgo.Session) negroni.Handler {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		s := session.Clone()
		defer s.Close()
		context.Set(r, "dbSession", s)
		context.Set(r, "DB", s.DB(DBNAME))
		next(w, r)
	})
}

func ParseFormMiddleware() negroni.Handler {

	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		err := r.ParseForm()
		if err != nil {
			ISR(w, r, err)
			return
		}
		next(w, r)
		if strings.Contains(r.Header.Get("Content-Type"), "multipart") {
			err = r.ParseMultipartForm(1024)
			if err != nil {
				ISR(w, r, err)
				return
			}
		}

	})

}
