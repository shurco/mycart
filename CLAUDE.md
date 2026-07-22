# Project Description

@AGENTS.md

# Developer Testing

use below command to test on development machine.
```bash
$ pushd web/admin/ && npx vite build && popd && pushd web/site/ && npx vite build && popd && go run ./cmd serve
```

