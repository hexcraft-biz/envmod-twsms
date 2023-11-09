package twsms

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/hexcraft-biz/her"
)

// ================================================================
//
// ================================================================
type Twsms struct {
	Username string
	Password string
	URL      *url.URL
}

func New() (*Twsms, her.Error) {
	u, err := url.Parse("https://api.twsms.com/json/sms_send.php")
	if err != nil {
		return nil, her.NewError(http.StatusInternalServerError, err, nil)
	}

	e := &Twsms{
		Username: os.Getenv("TWSMS_USERNAME"),
		Password: os.Getenv("TWSMS_PASSWORD"),
	}

	q := u.Query()
	q.Set("username", e.Username)
	q.Set("password", e.Password)
	u.RawQuery = q.Encode()
	e.URL = u

	return e, nil
}

type TwSmsSendApiResp struct {
	Code  string `json:"code"`
	Text  string `json:"text"`
	Msgid int64  `json:"msgid"`
}

func (r TwSmsSendApiResp) Error() her.Error {
	if code, err := strconv.Atoi(r.Code); err != nil {
		return her.NewError(http.StatusInternalServerError, err, nil)
	} else {
		switch {
		case code <= 1:
			return nil
		case code >= 10 && code <= 12:
			return her.NewErrorWithMessage(http.StatusInternalServerError, r.Text, nil)
		case code >= 50 && code <= 140:
			return her.NewErrorWithMessage(http.StatusInternalServerError, r.Text, nil)
		default:
			return her.NewErrorWithMessage(http.StatusServiceUnavailable, r.Text, nil)
		}
	}
}

func (e Twsms) SendSms(to []string, subject, body string) her.Error {
	if len(to) != 1 {
		return her.NewErrorWithMessage(http.StatusInternalServerError, "twsms module only support single reciever each request", nil)
	}

	if subject != "" {
		body = subject + body
	}

	q := e.URL.Query()
	q.Set("mobile", to[0])
	q.Set("message", body)
	e.URL.RawQuery = q.Encode()

	if req, err := http.NewRequest("POST", e.URL.String(), nil); err != nil {
		return her.NewError(http.StatusInternalServerError, err, nil)
	} else {
		client := &http.Client{}
		if resp, err := client.Do(req); err != nil {
			return her.NewError(http.StatusInternalServerError, err, nil)
		} else {
			defer resp.Body.Close()
			switch {
			case resp.StatusCode >= 500:
				return her.ErrServiceUnavailable
			case resp.StatusCode >= 400:
				return her.ErrInternalServerError
			}

			apiresp := new(TwSmsSendApiResp)
			decoder := json.NewDecoder(resp.Body)
			if err := decoder.Decode(apiresp); err != nil {
				return her.NewError(http.StatusInternalServerError, err, nil)
			} else {
				return apiresp.Error()
			}
		}
	}
}
