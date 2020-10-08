package instagram

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cute-angelia/downloader/library/config"
	"github.com/cute-angelia/go-utils/file"
	"github.com/guonaihong/gout"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

type instagram struct {
}

func NewInstagram() *instagram {
	return &instagram{
	}
}

var downloadUser = ""

func (self instagram) Do() {
	for _, v := range config.ConfigObj.Links {
		downloadUser = path.Base(v)
		log.Println(v, downloadUser)
		if err := getHomePage(v); err != nil {
			log.Println(err)
		}
	}
}

const (
	queryHash = "5b0222df65d7f6659c9b82246780caa7"
	queryURL  = "https://www.instagram.com/graphql/query"
)

func getHomePage(uri string) error {
	if rngInfo, err := createRangeInfo(config.ConfigObj.From, config.ConfigObj.To, config.ConfigObj.Offset, config.ConfigObj.Count); err != nil {
		log.Println(err)
	} else {
		body := ""
		gout.GET(uri).SetHeader(gout.H{
			"user-agent": config.ConfigObj.UserAgent,
			"cookie":     config.ConfigObj.Cookies,
		}).BindBody(&body).Do()

		// log.Println(body)
		if s, err := extractJSON(strings.NewReader(body)); err != nil {
			log.Println(err)
		} else {
			p := new(ProfilePostPage)
			if err := json.Unmarshal([]byte(s), p); err != nil {
				log.Println(err)
				return err
			}

			paths := make(chan string)
			errc := make(chan error, 1)
			done := make(chan struct{})
			defer close(done)
			const numWorkers = 10
			var wg sync.WaitGroup
			wg.Add(numWorkers)
			for i := 0; i < numWorkers; i++ {
				go func() {
					downloadResources(done, errc, paths)
					wg.Done()
				}()
			}

			switch {
			case len(p.EntryData.ProfilePage) > 0:
				err = scrapeProfilePage(rngInfo, paths, p)
			case len(p.EntryData.PostPage) > 0:
				err = scrapePostPage(paths, p)
			default:
				err = errors.New("instaget.scrapeImaghttp2es: unrecognized page type")
			}

			close(paths)
			wg.Wait()
			close(errc)
			if err != nil {
				return err
			}
			if err = <-errc; err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func scrapeImages(ri rangeInfo, u string) error {
	body := ""
	gout.GET(u).SetHeader(gout.H{
		"user-agent": config.ConfigObj.UserAgent,
		"cookie":     config.ConfigObj.Cookies,
	}).BindBody(&body).Do()

	readerBody := strings.NewReader(body)

	s, err := extractJSON(readerBody)
	if err != nil {
		return err
	}
	p := new(ProfilePostPage)
	err = json.Unmarshal([]byte(s), p)
	if err != nil {
		return err
	}

	log.Println(p)

	return nil
}

func extractJSON(r io.Reader) (string, error) {
	n, err := findJSONNode(r)
	if err != nil {
		return "", err
	}
	data := n.Data[:len(n.Data)-1]
	idx := strings.Index(n.Data, "{")
	if idx == -1 {
		return data, fmt.Errorf("malformed JSON data")
	}
	return data[idx:], nil
}

func findJSONNode(r io.Reader) (*html.Node, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var prevNode, snode *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		switch n.Type {
		case html.ElementNode:
			prevNode = n
		case html.TextNode:
			if prevNode.Data != "script" {
				return
			}
			for _, a := range prevNode.Attr {
				if a.Key != "type" {
					continue
				}
				if !strings.HasPrefix(n.Data, "window._sharedData") {
					continue
				}
				snode = n
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if snode == nil {
		return nil, fmt.Errorf("missing script element")
	}
	return snode, nil
}

func scrapeProfilePage(ri rangeInfo, paths chan<- string, p *ProfilePostPage) error {
	tm := &p.EntryData.ProfilePage[0].Graphql.User.EdgeOwnerToTimelineMedia
	// Handle the first page first as a special case, and then do
	// pagination requests.
	for i := range tm.Edges {
		n := &tm.Edges[i].Node
		switch ri.includes(n) {
		case cont:
			continue
		case outOfRange:
			return nil
		case inRange:
		}
		switch {
		case n.Typename == "GraphImage":
			paths <- n.DisplayURL
		case n.Typename == "GraphSidecar":
			resp, err := doShortcodeRequest(n.Shortcode)
			if err != nil {
				return err
			}
			edges := resp.Graphql.ShortcodeMedia.EdgeSidecarToChildren.Edges
			for i := range edges {
				paths <- edges[i].Node.DisplayResources[2].Src
			}
		case n.Typename == "GraphVideo":
			resp, err := doShortcodeRequest(n.Shortcode)
			if err != nil {
				return err
			}
			paths <- resp.Graphql.ShortcodeMedia.VideoURL
		default:
		}
	}
	var hasNext bool
	hasNext = tm.PageInfo.HasNextPage
	user := &p.EntryData.ProfilePage[0].Graphql.User
	endCursor := user.EdgeOwnerToTimelineMedia.PageInfo.EndCursor
	rhxGis := p.RhxGis
	for hasNext {
		resp, err := getNextPage(user.ID, endCursor, rhxGis)
		if err != nil {
			return err
		}
		urls, keepGoing := resp.listURLs(ri)
		for _, u := range urls {
			paths <- u
		}
		if !keepGoing {
			return nil
		}
		hasNext = resp.Data.User.EdgeOwnerToTimelineMedia.PageInfo.HasNextPage
		endCursor = resp.Data.User.EdgeOwnerToTimelineMedia.PageInfo.EndCursor
	}
	return nil
}

func scrapePostPage(paths chan<- string, p *ProfilePostPage) error {
	for _, u := range p.listURLs() {
		paths <- u
	}
	return nil
}

func doShortcodeRequest(shortcode string) (*ShortcodeQueryResponse, error) {
	uri := "https://www.instagram.com/p/" + shortcode + "/?__a=1"
	body := ""
	gout.GET(uri).SetHeader(gout.H{
		"user-agent": config.ConfigObj.UserAgent,
		"cookie":     config.ConfigObj.Cookies,
	}).BindBody(&body).Do()

	p := new(ShortcodeQueryResponse)
	dec := json.NewDecoder(strings.NewReader(body))
	if err := dec.Decode(p); err != nil {
		return nil, err
	}
	return p, nil
}

func getNextPage(id string, endCursor string, rhxGis string) (*PaginationQueryResponse, error) {
	type paginationQuery struct {
		ID    string `json:"id"`
		First int    `json:"first"`
		After string `json:"after"`
	}

	query := url.Values{}
	query.Add("query_hash", queryHash)
	pr := &paginationQuery{
		ID:    id,
		First: 12,
		After: endCursor,
	}
	b, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}
	query.Add("variables", string(b))
	u, err := url.Parse(queryURL)
	if err != nil {
		return nil, err
	}
	u.RawQuery = query.Encode()
	resp, err := xhr(u, query, rhxGis)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	qresp := new(PaginationQueryResponse)
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(qresp)
	if err != nil {
		return nil, err
	}
	return qresp, nil
}

func xhr(u *url.URL, query url.Values, rhxGis string) (resp *http.Response, err error) {
	u.RawQuery = query.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	signature := md5.Sum([]byte(fmt.Sprintf("%s:%s", rhxGis, query.Get("variables"))))
	// let the golang handle content-encoding automagically by not setting
	// an accept-encoding header.
	// req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", config.ConfigObj.UserAgent)
	req.Header.Set("x-instagram-gis", hex.EncodeToString(signature[:]))
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("instaget.xhr: %s", resp.Status)
	}
	return resp, nil
}

func downloadResources(done chan struct{}, errc chan<- error, urls <-chan string) {
	for u := range urls {
		if err := downloadFile(u); err != nil {
			select {
			case errc <- err:
			case <-done:
				break
			}
		}
		select {
		case <-done:
			break
		default:
		}
	}
}

func downloadFile(urlStr string) error {

	body := ""
	gout.GET(urlStr).SetHeader(gout.H{
		"user-agent": config.ConfigObj.UserAgent,
		"cookie":     config.ConfigObj.Cookies,
	}).BindBody(&body).Do()

	/// 下载路径
	name := file.MakeNameByTimeline( urlStr, "")
	downloadSavePath := getDownloadPath(name)

	// 创建目录
	os.MkdirAll(path.Dir(downloadSavePath), os.ModePerm)

	out, err := os.Create(downloadSavePath)

	defer out.Close()
	_, err = io.Copy(out, strings.NewReader(body))
	if err != nil {
		return err
	}

	//log.Println("download_path:" + downloadSavePath)
	log.Println("download_url:" + urlStr, downloadSavePath)

	return nil
}

func getDownloadPath(resPath string) string {
	return config.ConfigObj.Path + "/" + downloadUser + "/" + resPath
}
