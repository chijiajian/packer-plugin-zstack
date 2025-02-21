// Copyright (c) HashiCorp, Inc.

package jsonutils

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"zstack.io/zstack-sdk-go/pkg/errors"
	"zstack.io/zstack-sdk-go/pkg/util/sortedmap"
	"zstack.io/zstack-sdk-go/pkg/util/utils"
)

func addQueryStringSeg(body JSONObject, segs []TextNumber, val []string) (JSONObject, error) {
	if len(segs) == 0 {
		if len(val) == 1 {
			return NewString(val[0]), nil
		} else if len(val) > 1 {
			return NewStringArray(val), nil
		}
		return nil, errors.Wrap(ErrNilInputField, "empty value???")
	}
	if body == nil {
		if segs[0].isNumber && segs[0].number == 0 {
			body = NewArray()
		} else {
			body = NewDict()
		}
	}
	switch jbody := body.(type) {
	case *JSONDict:
		key := segs[0].String()
		if !jbody.Contains(key) {
			next, err := addQueryStringSeg(nil, segs[1:], val)
			if err != nil {
				return nil, errors.Wrapf(err, "addQueryStringSeg %s with %s fail", body, key)
			}
			jbody.Add(next, key)
		} else {
			next, err := jbody.Get(key)
			if err != nil {
				return nil, errors.Wrapf(err, "get jsondict %s with %s fail", body, key)
			}
			addQueryStringSeg(next, segs[1:], val)
		}
		return jbody, nil
	case *JSONArray:
		index := segs[0].number
		arrSize := int64(jbody.Size())
		if index < arrSize {
			next, err := jbody.GetAt(int(index))
			if err != nil {
				return nil, errors.Wrapf(err, "get jsonarray %s at %d fail", body, index)
			}
			addQueryStringSeg(next, segs[1:], val)
		} else if arrSize == index {
			// new
			next, err := addQueryStringSeg(nil, segs[1:], val)
			if err != nil {
				return nil, errors.Wrapf(err, "addQueryStringSeg %s at %d fail", body, index)
			}
			jbody.Add(next)
		} else {
			return nil, errors.Wrapf(ErrOutOfIndexRange, "index %d out of range", index)
		}
		return jbody, nil
	default:
		return nil, errors.Wrapf(ErrTypeMismatch, "invalid body %s and key %s", body, segs)
	}
}

func (th *JSONDict) parseQueryString(str string) error {
	m, err := url.ParseQuery(str)
	if err != nil {
		return errors.Wrap(err, "url.ParseQuery")
	}
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	segmentKeys := strings2stringSegments(keys)
	sort.Sort(segmentKeys)
	for _, segs := range segmentKeys {
		_, err := addQueryStringSeg(th, segs, m[segments2string(segs)])
		if err != nil {
			return errors.Wrap(err, "addQueryStringSeg")
		}
	}
	return nil
}

func ParseQueryString(str string) (JSONObject, error) {
	dict := NewDict()
	err := dict.parseQueryString(str)
	if err != nil {
		return nil, errors.Wrap(err, "dict.parseQueryString")
	}
	return dict, nil
}

func simpleQueryString(key, val string) string {
	if len(key) > 0 && len(val) > 0 {
		return url.QueryEscape(key) + "=" + url.QueryEscape(val)
	} else if len(val) > 0 {
		return url.QueryEscape(val)
	} else if len(key) > 0 {
		return url.QueryEscape(key)
	} else {
		return ""
	}
}

func (th *JSONValue) _queryString(key string) string {
	return simpleQueryString(key, "")
}

func (th *JSONString) _queryString(key string) string {
	return simpleQueryString(key, th.data)
}

func (th *JSONInt) _queryString(key string) string {
	return simpleQueryString(key, th.String())
}

func (th *JSONFloat) _queryString(key string) string {
	return simpleQueryString(key, th.String())
}

func (th *JSONBool) _queryString(key string) string {
	return simpleQueryString(key, th.String())
}

func (th *JSONArray) _queryString(key string) string {
	rets := make([]string, 0)
	for i, val := range th.data {
		k := strconv.FormatInt(int64(i), 10)
		if len(key) > 0 {
			k = key + "." + k
		}
		rets = append(rets, val._queryString(k))
	}
	return strings.Join(rets, "&")
}

func (th *JSONDict) _queryString(key string) string {
	rets := make([]string, 0)
	for iter := sortedmap.NewIterator(th.data); iter.HasMore(); iter.Next() {
		k, vinf := iter.Get()
		v := vinf.(JSONObject)
		if len(key) > 0 {
			k = key + "." + k
		}
		rets = append(rets, v._queryString(k))
	}
	return strings.Join(rets, "&")
}

func (th *JSONValue) QueryString() string {
	return th._queryString("")
}

func (th *JSONDict) QueryString() string {
	return th._queryString("")
}

func QueryBoolean(query JSONObject, key string, defVal bool) bool {
	if query == nil {
		return defVal
	}
	jsonVal, _ := query.Get(key)
	if jsonVal != nil {
		str, _ := jsonVal.GetString()
		return utils.ToBool(str)
	} else {
		return defVal
	}
}
