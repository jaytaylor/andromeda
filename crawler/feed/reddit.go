package feed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/db"
)

const RedditAPIURL = "https://www.reddit.com/r/%v/new.json"

type Reddit struct {
	*timestamped
	sub string
}

func NewReddit(dbClient db.Client, sub string) *Reddit {
	f := &Reddit{
		timestamped: newTimestamped(dbClient, fmt.Sprintf("reddit/%v", sub)),
		sub:         sub,
	}
	return f
}

func (f *Reddit) Refresh() ([]string, error) {
	last, err := f.last()
	if err != nil {
		return nil, err
	}
	body, err := f.pull()
	if err != nil {
		return nil, err
	}
	var hits redditResponse
	if err := json.Unmarshal(body, &hits); err != nil {
		log.Info("Failed to parse JSON: %v", string(body))
		return nil, err
	}
	if len(hits.Data.Children) == 0 {
		return nil, ErrEmptyResult
	}
	// Prune down to entries newer than last run.
	var pruned []redditChild
	if last != nil {
		pruned = []redditChild{}
		for _, child := range hits.Data.Children {
			if time.Unix(int64(child.Data.CreatedUtc), 0).After(*last) {
				pruned = append(pruned, child)
			}
		}
	} else {
		pruned = hits.Data.Children
	}
	if len(pruned) == 0 {
		log.Debug("no new results")
		return nil, nil
	}
	// Reduce text data to only new results.
	if body, err = json.Marshal(&pruned); err != nil {
		return nil, err
	}
	if err := f.mark(time.Unix(int64(pruned[0].Data.CreatedUtc), 0)); err != nil {
		return nil, err
	}
	possiblePkgs := findPackages(string(body))
	return possiblePkgs, nil
}

func (f *Reddit) pull() ([]byte, error) {
	u := fmt.Sprintf(RedditAPIURL, f.sub)
	resp, err := doRequest("", u, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil

}

// redditResponse thank you https://mholt.github.io/json-to-go/.
type redditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		Modhash  string        `json:"modhash"`
		Dist     int           `json:"dist"`
		Children []redditChild `json:"children"`
		After    string        `json:"after"`
		Before   interface{}   `json:"before"`
	} `json:"data"`
}

type redditChild struct {
	Kind string `json:"kind"`
	Data struct {
		ApprovedAtUtc              float64       `json:"approved_at_utc"`
		Subreddit                  string        `json:"subreddit"`
		Selftext                   string        `json:"selftext"`
		UserReports                []interface{} `json:"user_reports"`
		Saved                      bool          `json:"saved"`
		ModReasonTitle             interface{}   `json:"mod_reason_title"`
		Gilded                     int           `json:"gilded"`
		Clicked                    bool          `json:"clicked"`
		Title                      string        `json:"title"`
		LinkFlairRichtext          []interface{} `json:"link_flair_richtext"`
		SubredditNamePrefixed      string        `json:"subreddit_name_prefixed"`
		Hidden                     bool          `json:"hidden"`
		Pwls                       int           `json:"pwls"`
		LinkFlairCSSClass          interface{}   `json:"link_flair_css_class"`
		Downs                      int           `json:"downs"`
		ParentWhitelistStatus      string        `json:"parent_whitelist_status"`
		HideScore                  bool          `json:"hide_score"`
		Name                       string        `json:"name"`
		Quarantine                 bool          `json:"quarantine"`
		LinkFlairTextColor         string        `json:"link_flair_text_color"`
		AuthorFlairBackgroundColor interface{}   `json:"author_flair_background_color"`
		SubredditType              string        `json:"subreddit_type"`
		Ups                        int           `json:"ups"`
		Domain                     string        `json:"domain"`
		MediaEmbed                 struct {
		} `json:"media_embed"`
		AuthorFlairTemplateID interface{} `json:"author_flair_template_id"`
		IsOriginalContent     bool        `json:"is_original_content"`
		SecureMedia           interface{} `json:"secure_media"`
		IsRedditMediaDomain   bool        `json:"is_reddit_media_domain"`
		Category              interface{} `json:"category"`
		SecureMediaEmbed      struct {
		} `json:"secure_media_embed"`
		LinkFlairText        interface{}   `json:"link_flair_text"`
		CanModPost           bool          `json:"can_mod_post"`
		Score                int           `json:"score"`
		ApprovedBy           interface{}   `json:"approved_by"`
		Thumbnail            string        `json:"thumbnail"`
		Edited               bool          `json:"edited"`
		AuthorFlairCSSClass  interface{}   `json:"author_flair_css_class"`
		AuthorFlairRichtext  []interface{} `json:"author_flair_richtext"`
		ContentCategories    interface{}   `json:"content_categories"`
		IsSelf               bool          `json:"is_self"`
		ModNote              interface{}   `json:"mod_note"`
		Created              float64       `json:"created"`
		LinkFlairType        string        `json:"link_flair_type"`
		Wls                  int           `json:"wls"`
		PostCategories       interface{}   `json:"post_categories"`
		BannedBy             interface{}   `json:"banned_by"`
		AuthorFlairType      string        `json:"author_flair_type"`
		ContestMode          bool          `json:"contest_mode"`
		SelftextHTML         string        `json:"selftext_html"`
		Likes                interface{}   `json:"likes"`
		SuggestedSort        interface{}   `json:"suggested_sort"`
		BannedAtUtc          float64       `json:"banned_at_utc"`
		ViewCount            interface{}   `json:"view_count"`
		Archived             bool          `json:"archived"`
		NoFollow             bool          `json:"no_follow"`
		IsCrosspostable      bool          `json:"is_crosspostable"`
		Pinned               bool          `json:"pinned"`
		Over18               bool          `json:"over_18"`
		Media                interface{}   `json:"media"`
		MediaOnly            bool          `json:"media_only"`
		CanGild              bool          `json:"can_gild"`
		Spoiler              bool          `json:"spoiler"`
		Locked               bool          `json:"locked"`
		AuthorFlairText      interface{}   `json:"author_flair_text"`
		RteMode              string        `json:"rte_mode"`
		Visited              bool          `json:"visited"`
		NumReports           interface{}   `json:"num_reports"`
		Distinguished        interface{}   `json:"distinguished"`
		SubredditID          string        `json:"subreddit_id"`
		ModReasonBy          interface{}   `json:"mod_reason_by"`
		RemovalReason        interface{}   `json:"removal_reason"`
		ID                   string        `json:"id"`
		ReportReasons        interface{}   `json:"report_reasons"`
		Author               string        `json:"author"`
		NumCrossposts        int           `json:"num_crossposts"`
		NumComments          int           `json:"num_comments"`
		SendReplies          bool          `json:"send_replies"`
		AuthorFlairTextColor interface{}   `json:"author_flair_text_color"`
		Permalink            string        `json:"permalink"`
		WhitelistStatus      string        `json:"whitelist_status"`
		Stickied             bool          `json:"stickied"`
		URL                  string        `json:"url"`
		SubredditSubscribers int           `json:"subreddit_subscribers"`
		CreatedUtc           float64       `json:"created_utc"`
		ModReports           []interface{} `json:"mod_reports"`
		IsVideo              bool          `json:"is_video"`
	} `json:"data"`
}
