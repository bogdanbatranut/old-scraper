package notifications

import (
	"net/http"
	"old-scraper/pkg/dbmodels"
	"strings"
)

type NotificationsService struct {
	URL string
}

func NewNotificationService(url string) *NotificationsService {
	return &NotificationsService{URL: url}
}

func (s NotificationsService) PushCarNotification(car dbmodels.Car) {
	http.Post(s.URL, "text/plain",
		strings.NewReader("Service Old autovit STARTED 😀"))
}

func (s NotificationsService) PushTextNotification(text string) {
	http.Post(s.URL, "text/plain",
		strings.NewReader(text))
}

func (s NotificationsService) PushSuccessNotification(text string) {
	req, _ := http.NewRequest("POST", s.URL, strings.NewReader(text))
	req.Header.Set("Tags", "+1")
	http.DefaultClient.Do(req)
}

func (s NotificationsService) PushSuccessNotificationWithAction(text string, actionURL string) {
	req, _ := http.NewRequest("POST", s.URL, strings.NewReader(text))
	req.Header.Set("Tags", "+1")
	req.Header.Set("Click", actionURL)
	http.DefaultClient.Do(req)
}

func (s NotificationsService) PushErrNotification(text string) {
	req, _ := http.NewRequest("POST", s.URL, strings.NewReader(text))
	req.Header.Set("Tags", "warning")
	http.DefaultClient.Do(req)
}

func (s NotificationsService) PushErrRetryNotification(text string, retryURL string) {
	req, _ := http.NewRequest("POST", s.URL, strings.NewReader(text))
	req.Header.Set("Tags", "warning")
	req.Header.Set("Click", retryURL)
	http.DefaultClient.Do(req)
}
