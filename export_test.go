package planb

import "github.com/bsm/redeo/resp"

func RespondWith(w resp.ResponseWriter, v interface{}) { respondWith(w, v) }
