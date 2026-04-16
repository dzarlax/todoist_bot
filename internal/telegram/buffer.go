package telegram

import (
	"sync"
	"time"
)

// taskBuffer accumulates messages for a chat within a debounce window before
// they're turned into a single Todoist task.
type taskBuffer struct {
	messages []string
	priority int
	labels   []string
	timer    *time.Timer
}

// mediaGroupBuffer accumulates Telegram messages sharing the same MediaGroupID
// so an album can be processed as one task (like node-telegram-bot-api's
// `mediagroup` event).
type mediaGroupBuffer struct {
	messages []tgMessage // opaque to avoid import cycles in tests
	timer    *time.Timer
}

// tgMessage is an untyped alias — real type is *tgbotapi.Message. Declared
// separately to make the buffer file self-contained.
type tgMessage = interface{}

// bufferStore bundles both buffer maps under one mutex.
type bufferStore struct {
	mu     sync.Mutex
	tasks  map[int64]*taskBuffer
	groups map[string]*mediaGroupBuffer
}

func newBufferStore() *bufferStore {
	return &bufferStore{
		tasks:  map[int64]*taskBuffer{},
		groups: map[string]*mediaGroupBuffer{},
	}
}
