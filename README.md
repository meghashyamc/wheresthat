## Where's That?

This is supposed to be a simple search engine for local files. It's currently very much a work in progress.

There will be a web based UI that can be used to search for text a) within text files b) in file names and get search results the way we get in web search engines.

## Run

```sh
go install github.com/meghashyamc/wheresthat@latest
```
After running the binary, open `http://localhost:8080/ui` in a browser to index/search for files.

To change default config values, checkout `config/config.local.yaml` or set relevant environment variables (refer to `config/config.go`).
