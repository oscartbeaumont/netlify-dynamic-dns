package analytics

import (
	"net/http"
	"time"
)

const analyticsReportEndpoint = "https://plausible.otbeaumont.me/api/event"

var client = &http.Client{
	Timeout: time.Second * 10,
}

// Report an analytics event. The event is sent to a custom proxy for Simple Analytics which adds the users location based on their public IP because this is hard to do in native Go.
func Report(version string) error {
   /* if os.Getenv("NDDNS_DISABLE_ANALYTICS") == "true" {*/
		/*return nil*/
	/*}*/

	/*// Note: The OS versions are not updated as the only purpose of request not being reported as Bot and containing correct OS*/
	/*var userAgent = "Mozilla/5.0 "*/
	/*if runtime.GOOS == "windows" {*/
		/*userAgent += "(Windows NT 10.0; Win64; x64)"*/
	/*} else if runtime.GOOS == "darwin" {*/
		/*userAgent += "(Macintosh; Intel Mac OS X 11_1)"*/
	/*} else if runtime.GOOS == "linux" {*/
		/*userAgent += "(X11; Linux x86_64)"*/
	/*}*/

	/*var body = map[string]interface{}{*/
		/*"name":     "pageview",*/
		/*"url":      "https://nddns.app.otbeaumont.me/" + version,*/
		/*"domain":   "nddns.app.otbeaumont.me",*/
		/*"referrer": runtime.GOOS,*/
	/*}*/

	/*eventJSON, err := json.Marshal(body)*/
	/*if err != nil {*/
		/*return fmt.Errorf("error marshaling analytics event: %w", err)*/
	/*}*/

	/*req, err := http.NewRequest("POST", analyticsReportEndpoint, bytes.NewBuffer(eventJSON))*/
	/*if err != nil {*/
		/*return fmt.Errorf("error creating analytics event request: %w", err)*/
	/*}*/

	/*req.Header.Set("Content-Type", "application/json")*/
	/*req.Header.Set("User-Agent", userAgent)*/

	/*resp, err := client.Do(req)*/
	/*if err != nil {*/
		/*return fmt.Errorf("error posting analytics event: %w", err)*/
	/*} else if resp.StatusCode != http.StatusAccepted {*/
		/*return fmt.Errorf("error reported status %v with analytics event", resp.StatusCode)*/
	/*}*/

	return nil
}
