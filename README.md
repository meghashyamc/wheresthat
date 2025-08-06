## Where's That?

This is supposed to be a simple search engine for local files.

A web based UI can be used to search for text a) within text files b) in file names and get search results the way we get in web search engines.

## Run

```sh
go install github.com/meghashyamc/wheresthat@latest
```
After running the binary, open `http://localhost:8080/ui` in a browser to index/search for files.

To change default config values, checkout `config/config.local.yaml` or set relevant environment variables (refer to `config/config.go`).

Note: The UI has a button to index files. Files need to be indexed before they can be searched. If you're searching after a while, it's a good idea to index again. _Re-indexing is incremental and should be much faster than indexing the first time._