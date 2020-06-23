package watchdog

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return cli, s.Close
}

func TestGetWorkflowFile(t *testing.T) {

	httpClient, teardown := testingHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.String() == "/MyOrg/MyRepo/raw/12345678/.github/workflows/build.yaml" {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintln(w, `
			name: Build
			
			on:
			  push:
				## Always build when there are new commits to master
				#branches:
				#  - master
			
				# Always build release-tags
				tags:
				  - 'releases/**'
			
			jobs:
			  placeholder:
				name: "echo hello world, just to get started"
				runs-on: ubuntu-latest
				steps:
				  - run: echo hello && sleep 60 && echo world
			
			  build-win64:
				name: "Build for Win64"
			
				runs-on: build_agent
			
				timeout-minutes: 120
			
				steps:
				  - name: Check out repository
					uses: actions/checkout@v2
					with:
					  clean: false
			
				  - name: Setup credentials for cloud storage
					run: $env:LONGTAIL_GCLOUD_CREDENTIALS | Out-File FetchPrebuiltUE4\application-default-credentials.json -Encoding ASCII
					env:
					  LONGTAIL_GCLOUD_CREDENTIALS: ${{ secrets.LONGTAIL_GCLOUD_CREDENTIALS }}
			
				  - name: Update UE4
					run: .\UpdateUE4.bat
			
				  - name: Build game (Win64)
					run: .\BuildGame.bat
			
				  - name: Upload game as Game-${{ github.sha }}
					run: .\UploadGame ${{ github.sha }}
			`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer teardown()

	u, _ := url.Parse("http://example.com")

	gitHubSite := &GitHubApiSite{BaseUrl: *u, Client: httpClient}

	t.Run("Fetch workflow file that exists", func(t *testing.T) {

		_, err := getWorkflowFile(gitHubSite, "MyOrg", "MyRepo", "12345678", ".github/workflows/build.yaml")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Fetch workflow file that does not exist", func(t *testing.T) {

		_, err := getWorkflowFile(gitHubSite, "MyOrg2", "MyRepo2", "12345679", ".github/workflows/build.yaml")
		if err == nil {
			t.Fatal("Should have failed")
		}
	})
}