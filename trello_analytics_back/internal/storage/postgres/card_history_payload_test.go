package postgres

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseHistoryPayload_ListMoveAndMember(t *testing.T) {
	payload := json.RawMessage(`{
		"idMemberCreator": "mem1",
		"data": {
			"listBefore": {"id": "l1", "name": "В работе"},
			"listAfter": {"id": "l2", "name": "Готово"},
			"card": {"id": "c1"}
		}
	}`)
	meta := parseHistoryPayload(payload)
	if meta.MemberTrelloID != "mem1" {
		t.Fatalf("member=%q", meta.MemberTrelloID)
	}
	if meta.SourceListTrelloID != "l1" || meta.SourceListName != "В работе" {
		t.Fatalf("source=%+v", meta)
	}
	if meta.DestListTrelloID != "l2" || meta.DestListName != "Готово" {
		t.Fatalf("dest=%+v", meta)
	}
}

func TestParseHistoryPayload_CreateCardUsesList(t *testing.T) {
	payload := json.RawMessage(`{
		"idMemberCreator": "mem2",
		"data": {
			"list": {"id": "l9", "name": "Бэклог"},
			"card": {"id": "c9"}
		}
	}`)
	meta := parseHistoryPayload(payload)
	if meta.SourceListName != "" || meta.SourceListTrelloID != "" {
		t.Fatalf("create should have empty source: %+v", meta)
	}
	if meta.DestListTrelloID != "l9" || meta.DestListName != "Бэклог" {
		t.Fatalf("dest=%+v", meta)
	}
	if meta.MemberTrelloID != "mem2" {
		t.Fatalf("member=%q", meta.MemberTrelloID)
	}
}

func TestCardHistoryItem_JSONIncludesDisplayNames(t *testing.T) {
	name := "Ada"
	src := "Doing"
	dst := "Done"
	item := CardHistoryItem{
		ID: "a1", TrelloActionID: "t1", ActionType: "updateCard",
		MemberName: &name, SourceListName: &src, DestinationListName: &dst,
	}
	b, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"member_name"`, `"source_list_name"`, `"destination_list_name"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
}
