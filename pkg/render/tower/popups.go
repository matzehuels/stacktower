package tower

import (
	"bytes"
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/styles"
)

const (
	popupCSS = `
    .popup { pointer-events: none; transition: opacity 0.15s ease, transform 0.1s ease; }
    .popup[visibility="hidden"] { opacity: 0; }
    .popup[visibility="visible"] { opacity: 1; }`

	popupJS = `
    const svg = document.querySelector('svg');
    const vb = svg.viewBox.baseVal;
    document.querySelectorAll('.block-text').forEach(el => {
      const id = el.dataset.block;
      const popup = document.querySelector('.popup[data-for="' + id + '"]');
      if (!popup) return;
      el.style.cursor = 'pointer';
      el.addEventListener('mouseenter', () => {
        const textBox = el.getBBox();
        const popupBox = popup.getBBox();
        let x = textBox.x + textBox.width/2 - popupBox.width/2;
        let y = textBox.y + textBox.height + 12;
        if (y + popupBox.height > vb.y + vb.height - 10) y = textBox.y - popupBox.height - 8;
        if (y < vb.y + 10) y = vb.y + 10;
        x = Math.max(vb.x + 10, Math.min(x, vb.x + vb.width - popupBox.width - 10));
        popup.setAttribute('transform', 'translate(' + x.toFixed(1) + ',' + y.toFixed(1) + ')');
        popup.setAttribute('visibility', 'visible');
      });
      el.addEventListener('mouseleave', () => popup.setAttribute('visibility', 'hidden'));
    });`
)

func extractPopupData(n *dag.Node) *styles.PopupData {
	if n.Meta == nil {
		return nil
	}
	p := &styles.PopupData{
		Stars:       asInt(n.Meta["repo_stars"]),
		Maintainers: countMaintainers(n.Meta["repo_maintainers"]),
		Brittle:     IsBrittle(n),
	}
	p.LastCommit, _ = n.Meta["repo_last_commit"].(string)
	p.LastRelease, _ = n.Meta["repo_last_release"].(string)
	p.Archived, _ = n.Meta["repo_archived"].(bool)

	if desc, ok := n.Meta["description"].(string); ok && desc != "" {
		p.Description = desc
	} else if summary, ok := n.Meta["summary"].(string); ok && summary != "" {
		p.Description = summary
	}
	return p
}

func asInt(v any) int {
	switch v := v.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func RenderPopupScript(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "  <style>%s\n  </style>\n", popupCSS)
	fmt.Fprintf(buf, "  <script type=\"text/javascript\"><![CDATA[%s\n  ]]></script>\n", popupJS)
}
