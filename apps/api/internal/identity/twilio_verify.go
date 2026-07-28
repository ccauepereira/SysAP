package identity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const twilioVerifyBaseURL = "https://verify.twilio.com/v2/Services/"
const maxTwilioBody = 64 << 10

var e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var ErrVerificationUnavailable = errors.New("verification unavailable")

type VerifyProvider interface {
	Start(context.Context, string) error
	Check(context.Context, string, string) (bool, error)
}
type TwilioVerifyProvider struct {
	accountSID, authToken, serviceSID string
	client                            *http.Client
}

func NewTwilioVerifyProvider(accountSID, authToken, serviceSID string) (*TwilioVerifyProvider, error) {
	if accountSID == "" || authToken == "" || serviceSID == "" {
		return nil, ErrVerificationUnavailable
	}
	return &TwilioVerifyProvider{accountSID: accountSID, authToken: authToken, serviceSID: serviceSID, client: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (p *TwilioVerifyProvider) Start(ctx context.Context, phone string) error {
	_, e := p.call(ctx, "Verifications", url.Values{"To": {phone}, "Channel": {"sms"}})
	return e
}
func (p *TwilioVerifyProvider) Check(ctx context.Context, phone, code string) (bool, error) {
	b, e := p.call(ctx, "VerificationCheck", url.Values{"To": {phone}, "Code": {code}})
	if e != nil {
		return false, e
	}
	var r struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(b, &r) != nil {
		return false, ErrVerificationUnavailable
	}
	return r.Status == "approved", nil
}
func (p *TwilioVerifyProvider) call(ctx context.Context, path string, form url.Values) ([]byte, error) {
	if !e164Pattern.MatchString(form.Get("To")) {
		return nil, ErrVerificationUnavailable
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, twilioVerifyBaseURL+p.serviceSID+"/"+path, strings.NewReader(form.Encode()))
	if e != nil {
		return nil, ErrVerificationUnavailable
	}
	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, e := p.client.Do(req)
	if e != nil {
		return nil, ErrVerificationUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, ErrVerificationUnavailable
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, maxTwilioBody+1))
	if e != nil || len(b) == 0 || len(b) > maxTwilioBody {
		return nil, ErrVerificationUnavailable
	}
	return b, nil
}
