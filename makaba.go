package makaba

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strings"
)

const (
	domain     = "2ch.hk"
	makabaUrl  = "https://2ch.hk/makaba/makaba.fcgi"
	postingUrl = "https://2ch.hk/makaba/posting.fcgi?json=1"
)

type Request *http.Request
type Response *http.Response

// A SuperAgent is an object storing all required request data
type SuperAgent struct {
	Passcode string
	Cookies  []*http.Cookie
	BodyData map[string]string
	FileData []FileData
	Client   *http.Client
	Errors   []error
}

// Here we store minimal base fields like board, thread, name, email, subject and comment to make a post
type BodyData struct {
	Key   string
	Value string
}

// For file attachments like .jpg, .webm .mp4, etc to upload with your post
type FileData struct {
	Filename  string
	Fieldname string
	Data      []byte
}

//
type GetData struct {
	Board       string
	CatalogJSON CatalogJSON
	Num         int
	Errors      []error
}

type CatalogJSON struct {
	Board     string `json:"board"`
	BoardInfo string `json:"BoardInfo"`
	Threads   []struct {
		Comment    string      `json:"comment"`
		Lasthit    int         `json:"lasthit"`
		Num        string      `json:"num"`
		PostsCount int         `json:"posts_count"`
		Score      json.Number `json:"score"`
		Subject    string      `json:"subject"`
		Timestamp  int         `json:"timestamp"`
		Views      int         `json:"views"`
	} `json:"threads"`
}

// Used to create a new SuperAgent object
func Post() *SuperAgent {

	p := &SuperAgent{
		Passcode: "",
		Client:   &http.Client{},
		BodyData: make(map[string]string),
		FileData: make([]FileData, 0),
	}
	return p
}

// Used to authorize passcode into usercode to use it later with HTTP cookie to bypass CAPTCHA. Details https://2ch.hk/2ch
func (p *SuperAgent) PasscodeAuth(passcode string) bool {
	formData := url.Values{
		"json":     {"1"},
		"task":     {"auth"},
		"usercode": {passcode},
	}

	resp, err := http.PostForm(makabaUrl, formData)
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false
	}

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Println(err)
		return false
	}

	// If failed
	if result["result"].(float64) == 0 {
		log.Printf("❌ Passcode auth failed: %v", result)
		return false
	}

	// If succeed
	if result["result"].(float64) == 1 {
		hash := fmt.Sprint(result["hash"])
		//log.Printf("✅ Passcode auth succeed: %v", hash)
		p.Passcode = hash
		jar, _ := cookiejar.New(nil)
		var cookies []*http.Cookie
		cookie := &http.Cookie{
			Name:   "passcode_auth",
			Value:  p.Passcode,
			Path:   "/",
			Domain: "2ch.hk",
		}
		cookies = append(cookies, cookie)
		u, _ := url.Parse(postingUrl)
		jar.SetCookies(u, cookies)
		p.Client = &http.Client{
			Jar: jar,
		}
		return true
	}

	return false
}

// Used to set board field. Required.
func (p *SuperAgent) Board(board string) *SuperAgent {
	p.BodyData["task"] = "post"
	p.BodyData["board"] = board
	return p
}

// Used to set thread number field. Required. Leave it empty or set 0 to create new thread
func (p *SuperAgent) Thread(thread string) *SuperAgent {
	p.BodyData["thread"] = thread
	return p
}

// Used to set name field
func (p *SuperAgent) Name(name string) *SuperAgent {
	p.BodyData["name"] = ""
	return p
}

// Used to set email field
func (p *SuperAgent) Mail(mail string) *SuperAgent {
	p.BodyData["email"] = mail
	return p
}

// Used to set subject field
func (p *SuperAgent) Subject(subject string) *SuperAgent {
	p.BodyData["subject"] = subject
	return p
}

// Used to set comment field, this is where your post text goes
func (p *SuperAgent) Comment(comment string) *SuperAgent {
	p.BodyData["comment"] = comment
	return p
}

// File() accepts string as path to a local file or remote HTTP URL.
// Example: File("1.png") or File("https://i.imgur.com/1.png") or File("1.jpg", "https://i.imgur.com/1.png")
func (p *SuperAgent) File(file ...interface{}) *SuperAgent {

	filename := "123"
	fieldname := "file"

	for _, file := range file {
		log.Println(file)

		switch v := reflect.ValueOf(file); v.Kind() {
		case reflect.String:
			// Check for URL string
			if u, err := url.Parse(v.String()); err == nil && u.Scheme != "" && u.Host != "" {
				log.Println("true")

				resp, err := http.Get(v.String())
				if err != nil {
					log.Println("Failed to fetch data from HTTP URL")
				}

				data, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Println("Failed to io.ReadAll: %v", err)
				}

				p.FileData = append(p.FileData, FileData{
					Filename:  filename,
					Fieldname: fieldname,
					Data:      data,
				})
				continue
			}
			data, err := ioutil.ReadFile(v.String())
			if err != nil {
				p.Errors = append(p.Errors, err)
				continue
			}

			p.FileData = append(p.FileData, FileData{
				Filename:  filename,
				Fieldname: fieldname,
				Data:      data,
			})
			continue
		default:
			if v.Type() == reflect.TypeOf(os.File{}) {
				osfile := v.Interface().(os.File)

				data, err := ioutil.ReadFile(osfile.Name())
				if err != nil {
					p.Errors = append(p.Errors, err)
					//return p
					continue
				}

				p.FileData = append(p.FileData, FileData{
					Filename:  filename,
					Fieldname: fieldname,
					Data:      data,
				})
				continue
			}
		}
	}
	return p
}

func (p *SuperAgent) getResponseBytes() (Response, []byte, []error) {
	var (
		req  *http.Request
		err  error
		resp Response
	)

	req, err = p.MakeRequest()
	if err != nil {
		p.Errors = append(p.Errors, err)
		return nil, nil, p.Errors
	}

	// Send request
	resp, err = p.Client.Do(req)
	if err != nil {
		p.Errors = append(p.Errors, err)
		return nil, nil, p.Errors
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, []error{err}
	}

	return resp, body, nil
}

func (p *SuperAgent) MakeRequest() (*http.Request, error) {
	var (
		req           *http.Request
		contentReader io.Reader
		err           error
		buf           = &bytes.Buffer{}
		mw            = multipart.NewWriter(buf)
	)

	if len(p.BodyData) != 0 {
		for k, v := range p.BodyData {
			fw, _ := mw.CreateFormField(k)
			fw.Write([]byte(v))
			//            io.Copy(fw, strings.NewReader(v))
		}
	}

	if len(p.FileData) != 0 {
		for _, file := range p.FileData {
			fw, _ := mw.CreateFormFile(file.Fieldname, "")
			fw.Write(file.Data)
		}

	}
	mw.Close()

	contentReader = buf
	if req, err = http.NewRequest("POST", postingUrl, contentReader); err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", mw.FormDataContentType())

	return req, nil
}

// Do() is the most important function that you need to call when ending the chain. The request won't proceed without calling it.
func (p *SuperAgent) Do(passcode string) (string, error) {
	var (
		resp *http.Response
		errs []error
		body []byte
	)

	p.PasscodeAuth(passcode)

	resp, body, errs = p.getResponseBytes()
	for _, e := range errs {
		if e != nil {
			return "", e
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Println(err)
		return "", err
	}

	// If failed
	if v, found := result["Error"].(float64); found {
		log.Printf("❌ Posting failed: %v, %v", result, v)
		return "", fmt.Errorf("%v", result)
	} else {
		// If succeed
		log.Printf("✔ Posting succeed: %v", result)
		if result["Status"].(string) == "Redirect" {
			num := fmt.Sprintf("%.0f", result["Target"].(float64))
			return num, nil
		}
		num := fmt.Sprintf("%.0f", result["Num"].(float64))
		return num, nil
	}

	return "", fmt.Errorf("❌ Posting failed")
}

// Used to create a new Catalog object
func Get(board string) *GetData {
	if board == "" {
		log.Println("You have to specify board")
	}

	g := &GetData{
		CatalogJSON: CatalogJSON{},
		Board:       board,
		Num:         0,
	}

	return g
}

func (g *GetData) Catalog(params ...string) *GetData {
	var sort string

	if len(params) > 1 {
		log.Println("Options can only be top, bump, date or empty")
		return g
	}
	if len(params) == 1 {
		sort = params[0]
	}
	switch sort {
	case "bump":
		url := fmt.Sprintf("https://%v/%v/catalog.json", domain, g.Board)
		body := g.getResponseBytes(url)
		err := json.Unmarshal(body, &g.CatalogJSON)
		if err != nil {
			log.Println(err)
		}
		return g
	case "date":
		url := fmt.Sprintf("https://%v/%v/catalog_num.json", domain, g.Board)
		body := g.getResponseBytes(url)
		err := json.Unmarshal(body, &g.CatalogJSON)
		if err != nil {
			log.Println(err)
		}
		return g
	default:
		url := fmt.Sprintf("https://%v/%v/threads.json", domain, g.Board)
		body := g.getResponseBytes(url)
		err := json.Unmarshal(body, &g.CatalogJSON)
		if err != nil {
			log.Println(err)
		}
		return g
	}
	return g
}

func (g *GetData) getResponseBytes(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	return body
}

func (g *GetData) Thread(keyword string) (string, string, error) {
	g.Catalog()
	//log.Println(threads)
	for _, v := range g.CatalogJSON.Threads {
		//log.Println(k, v.Num, v.Subject)
		if strings.Contains(strings.ToLower(v.Subject), strings.ToLower(keyword)) {
			//log.Println(k, v.Num, v.Subject)
			return v.Num, v.Subject, nil
		}
	}
	return "", "", fmt.Errorf("Couldn't find any threads matching this \"%v\" keyword", keyword)
}
