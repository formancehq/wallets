// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type ServerInfo struct {
	Version string `json:"version"`
}

func (o *ServerInfo) GetVersion() string {
	if o == nil {
		return ""
	}
	return o.Version
}
