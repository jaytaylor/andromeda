package feed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/db"
)

var HackerNewsAPIURL = "https://hn.algolia.com/api/v1/search_by_date?hitsPerPage=%v&page=0&tags=story"

var (
	HackerNewsPerPage = 1000
)

type HackerNews struct {
	*timestamped
}

func NewHackerNews(dbClient db.Client) *HackerNews {
	f := &HackerNews{
		timestamped: newTimestamped(dbClient, "hackernews"),
	}
	return f
}

func (ds *HackerNews) Refresh() ([]string, error) {
	last, err := ds.last()
	if err != nil {
		return nil, err
	}
	body, err := ds.pull(last)
	if err != nil {
		return nil, err
	}
	var hits hackerNewsResponse
	if err := json.Unmarshal(body, &hits); err != nil {
		return nil, err
	}
	if len(hits.Hits) == 0 {
		return nil, ErrEmptyResult
		// Extract most recent timestamp.
		// 2018-07-20T03:50:44.000Z
	}
	//const layout = "2006-01-02T15:04:05.000Z"
	if err := ds.mark(hits.Hits[0].CreatedAt); err != nil {
		return nil, err
	}
	possiblePkgs := findPackages(string(body))
	return possiblePkgs, nil
}

func (ds *HackerNews) pull(since *time.Time) ([]byte, error) {
	u := fmt.Sprintf(HackerNewsAPIURL, HackerNewsPerPage)
	if since != nil {
		u = fmt.Sprintf("%v&numericFilters=created_at_i>%v", u, since.Unix())
	}
	log.WithField("url", u).Debug("Fetching page")
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

type hackerNewsResponse struct {
	Hits             []hackerNewsHit `json:"hits"`
	NbHits           int             `json:"nbHits"`
	Page             int             `json:"page"`
	NbPages          int             `json:"nbPages"`
	HitsPerPage      int             `json:"hitsPerPage"`
	ProcessingTimeMS int             `json:"processingTimeMS"`
	ExhaustiveNbHits bool            `json:"exhaustiveNbHits"`
	Query            string          `json:"query"`
	Params           string          `json:"params"`
}

type hackerNewsHit struct {
	CreatedAt       time.Time   `json:"created_at"`
	Title           interface{} `json:"title"`
	URL             interface{} `json:"url"`
	Author          string      `json:"author"`
	Points          interface{} `json:"points"`
	StoryText       interface{} `json:"story_text"`
	CommentText     string      `json:"comment_text"`
	NumComments     interface{} `json:"num_comments"`
	StoryID         int         `json:"story_id"`
	StoryTitle      string      `json:"story_title"`
	StoryURL        string      `json:"story_url"`
	ParentID        int         `json:"parent_id"`
	CreatedAtI      int         `json:"created_at_i"`
	Tags            []string    `json:"_tags"`
	ObjectID        string      `json:"objectID"`
	HighlightResult struct {
		Author struct {
			Value        string        `json:"value"`
			MatchLevel   string        `json:"matchLevel"`
			MatchedWords []interface{} `json:"matchedWords"`
		} `json:"author"`
		CommentText struct {
			Value        string        `json:"value"`
			MatchLevel   string        `json:"matchLevel"`
			MatchedWords []interface{} `json:"matchedWords"`
		} `json:"comment_text"`
		StoryTitle struct {
			Value        string        `json:"value"`
			MatchLevel   string        `json:"matchLevel"`
			MatchedWords []interface{} `json:"matchedWords"`
		} `json:"story_title"`
		StoryURL struct {
			Value        string        `json:"value"`
			MatchLevel   string        `json:"matchLevel"`
			MatchedWords []interface{} `json:"matchedWords"`
		} `json:"story_url"`
	} `json:"_highlightResult"`
}
