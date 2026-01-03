package models

import (
	"strings"
	"testing"
)

func TestCompileMJML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		contains []string
	}{
		{
			name: "simple mjml",
			input: `<mjml>
				<mj-body>
					<mj-section>
						<mj-column>
							<mj-text>Hello World</mj-text>
						</mj-column>
					</mj-section>
				</mj-body>
			</mjml>`,
			wantErr:  false,
			contains: []string{"Hello World", "<html", "<body", "<table"},
		},
		{
			name: "mjml with button",
			input: `<mjml>
				<mj-body>
					<mj-section>
						<mj-column>
							<mj-button href="https://example.com">Click Me</mj-button>
						</mj-column>
					</mj-section>
				</mj-body>
			</mjml>`,
			wantErr:  false,
			contains: []string{"Click Me", "https://example.com"},
		},
		{
			name: "mjml with image",
			input: `<mjml>
				<mj-body>
					<mj-section>
						<mj-column>
							<mj-image src="https://example.com/image.jpg" />
						</mj-column>
					</mj-section>
				</mj-body>
			</mjml>`,
			wantErr:  false,
			contains: []string{"<img", "https://example.com/image.jpg"},
		},
		{
			name:     "invalid mjml",
			input:    `<not-mjml>invalid</not-mjml>`,
			wantErr:  true,
			contains: nil,
		},
		{
			name:     "empty input",
			input:    ``,
			wantErr:  true,
			contains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompileMJML(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileMJML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, s := range tt.contains {
					if !strings.Contains(got, s) {
						t.Errorf("CompileMJML() output should contain %q, got %q", s, got)
					}
				}
			}
		})
	}
}

func TestCompileMJMLWithTemplateVariables(t *testing.T) {
	// Test that Go template variables are preserved after MJML compilation
	input := `<mjml>
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>Hello {{ .Subscriber.Name }}</mj-text>
				</mj-column>
			</mj-section>
		</mj-body>
	</mjml>`

	got, err := CompileMJML(input)
	if err != nil {
		t.Errorf("CompileMJML() error = %v", err)
		return
	}

	// Template variables should be preserved in the output
	if !strings.Contains(got, "{{ .Subscriber.Name }}") {
		t.Error("CompileMJML() should preserve Go template variables")
	}
}
