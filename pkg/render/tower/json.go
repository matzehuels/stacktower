package tower

import "encoding/json"

func RenderJSON(layout Layout) ([]byte, error) {
	return json.MarshalIndent(layout, "", "  ")
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
