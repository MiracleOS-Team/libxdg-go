package notificationDaemon

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

// Config allows customization of the daemon.
type Config struct {
	// LockFilePath is used for the file lock.
	// If empty, it defaults to $XDG_RUNTIME_DIR/notificationdaemon.lock or /tmp/notificationdaemon.lock.
	LockFilePath string
	// You can add additional customization options here.
	Capabilities []string
}

// Notification represents a notification event.
type Notification struct {
	ID            uint32
	AppName       string
	AppIcon       string
	Summary       string
	Body          string
	Actions       []string
	Hints         map[string]dbus.Variant
	ExpireTimeout int32
	Timestamp     time.Time
}

type NotificationEvent struct {
	Notification Notification
	Created      bool
	Modified     bool
	Deleted      bool
}

// Daemon implements the org.freedesktop.Notifications interface.
type Daemon struct {
	config               Config
	lockFile             *os.File
	conn                 *dbus.Conn
	mu                   sync.Mutex
	Notifications        map[uint32]Notification
	nextID               uint32
	NotificationsChannel chan NotificationEvent
	Logger               slog.Logger
}

// NewDaemon creates a new NotificationDaemon instance.
func NewDaemon(config Config) *Daemon {
	if config.LockFilePath == "" {
		xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
		if xdgRuntime == "" {
			xdgRuntime = os.TempDir()
		}
		config.LockFilePath = fmt.Sprintf("%s/notificationdaemon.lock", xdgRuntime)
	}
	return &Daemon{
		config:               config,
		Notifications:        make(map[uint32]Notification),
		nextID:               1,
		NotificationsChannel: make(chan NotificationEvent, 10),
		Logger:               *slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

// fileLock acquires an exclusive lock on the specified file.
func (d *Daemon) fileLock() error {
	f, err := os.OpenFile(d.config.LockFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return errors.New("another instance is already running")
	}
	d.lockFile = f
	return nil
}

// fileUnlock releases the file lock.
func (d *Daemon) fileUnlock() {
	if d.lockFile != nil {
		syscall.Flock(int(d.lockFile.Fd()), syscall.LOCK_UN)
		d.lockFile.Close()
		os.Remove(d.config.LockFilePath)
		d.lockFile = nil
	}
}

// Start initializes the DBus connection and registers the Notifications service.
func (d *Daemon) Start() error {
	// Acquire file lock.
	if err := d.fileLock(); err != nil {
		return err
	}

	// Connect to the session bus.
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		d.fileUnlock()
		return err
	}
	d.conn = conn

	// Request the well-known name "org.freedesktop.Notifications" on the bus.
	reply, err := d.conn.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue)
	if err != nil {
		d.fileUnlock()
		return err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		d.fileUnlock()
		return errors.New("notification daemon is already running (bus name taken)")
	}

	// Export the daemon object on the bus.
	err = d.conn.Export(d, "/org/freedesktop/Notifications", "org.freedesktop.Notifications")
	if err != nil {
		d.fileUnlock()
		return err
	}

	// Export introspection data for clients to inspect our interface.
	node := &introspect.Node{
		Name: "/org/freedesktop/Notifications",
		Interfaces: []introspect.Interface{
			{
				Name: "org.freedesktop.Notifications",
				Methods: []introspect.Method{
					{
						Name: "Notify",
						Args: []introspect.Arg{
							{Name: "app_name", Type: "s", Direction: "in"},
							{Name: "replaces_id", Type: "u", Direction: "in"},
							{Name: "app_icon", Type: "s", Direction: "in"},
							{Name: "summary", Type: "s", Direction: "in"},
							{Name: "body", Type: "s", Direction: "in"},
							{Name: "actions", Type: "as", Direction: "in"},
							{Name: "hints", Type: "a{sv}", Direction: "in"},
							{Name: "expire_timeout", Type: "i", Direction: "in"},
							{Name: "id", Type: "u", Direction: "out"},
						},
					},
					{
						Name: "CloseNotification",
						Args: []introspect.Arg{
							{Name: "id", Type: "u", Direction: "in"},
						},
					},
					{
						Name: "GetCapabilities",
						Args: []introspect.Arg{
							{Name: "capabilities", Type: "as", Direction: "out"},
						},
					},
					{
						Name: "GetServerInformation",
						Args: []introspect.Arg{
							{Name: "name", Type: "s", Direction: "out"},
							{Name: "vendor", Type: "s", Direction: "out"},
							{Name: "version", Type: "s", Direction: "out"},
							{Name: "spec_version", Type: "s", Direction: "out"},
						},
					},
				},
				Properties: []introspect.Property{},
				Signals: []introspect.Signal{
					{
						Name: "NotificationClosed",
						Args: []introspect.Arg{
							{Name: "id", Type: "u", Direction: "in"},
							{Name: "reason", Type: "u", Direction: "in"},
						},
					},
					{
						Name: "ActionInvoked",
						Args: []introspect.Arg{
							{Name: "id", Type: "u", Direction: "in"},
							{Name: "action_key", Type: "s", Direction: "in"},
						},
					},
				},
			},
			introspect.IntrospectData,
		},
	}
	err = d.conn.Export(introspect.NewIntrospectable(node), "/org/freedesktop/Notifications", "org.freedesktop.DBus.Introspectable")
	if err != nil {
		d.fileUnlock()
		return err
	}

	slog.Info("Notification daemon started on DBus as org.freedesktop.Notifications")
	return nil
}

// Stop shuts down the daemon.
func (d *Daemon) Stop() {
	if d.conn != nil {
		d.conn.Close()
	}
	d.fileUnlock()
}

// GetServerInformation returns static information about the notification server.
func (d *Daemon) GetServerInformation() (string, string, string, string, *dbus.Error) {
	// Customize these values as desired.
	return "libxdg-go notification daemon", "MiracleOS-Team", "1.1", "1.2", nil
}

// GetCapabilities returns the capabilities supported by the notification server.
func (d *Daemon) GetCapabilities() ([]string, *dbus.Error) {
	// Example capabilities; adjust to your implementation.
	caps := []string{"body", "actions"}
	return caps, nil
}

// Notify implements the Notify method as defined in the Desktop Notifications spec.
// It creates (or replaces) a notification and returns its ID.
func (d *Daemon) Notify(appName string, replacesID uint32, appIcon string, summary string, body string, actions []string, hints map[string]dbus.Variant, expireTimeout int32) (uint32, *dbus.Error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Use the provided replacesID if valid.
	id := replacesID
	if id == 0 || d.Notifications[id].ID == 0 {
		id = d.nextID
		d.nextID++
	}

	notification := Notification{
		ID:            id,
		AppName:       appName,
		AppIcon:       appIcon,
		Summary:       summary,
		Body:          body,
		Actions:       actions,
		Hints:         hints,
		ExpireTimeout: expireTimeout,
		Timestamp:     time.Now(),
	}
	d.Notifications[id] = notification

	// In a complete daemon, you might display the notification in a UI,
	// forward it to another handler, or log it.

	slog.Debug(strings.Join([]string{"Received notification ", strconv.Itoa(int(id)), ": ", summary, " - ", body}, "\n"))

	notificationEvent := NotificationEvent{
		Notification: notification,
		Created:      replacesID == 0,
		Modified:     replacesID != 0,
		Deleted:      false,
	}

	d.NotificationsChannel <- notificationEvent

	return id, nil
}

func (d *Daemon) InvokeAction(id uint32, action_key string) {
	d.conn.Emit(dbus.ObjectPath("/org/freedesktop/Notifications"), "org.freedesktop.Notifications.ActionInvoked", id, action_key)
}

// CloseNotification implements the CloseNotification method.
func (d *Daemon) CloseNotification(id uint32) *dbus.Error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.Notifications[id]; exists {

		d.conn.Emit(dbus.ObjectPath("/org/freedesktop/Notifications"), "org.freedesktop.Notifications.NotificationClosed", id, 3)
		slog.Debug(strings.Join([]string{"User closed notification ", strconv.Itoa(int(id))}, "\n"))

		notificationEvent := NotificationEvent{
			Notification: d.Notifications[id],
			Created:      false,
			Modified:     false,
			Deleted:      true,
		}
		delete(d.Notifications, id)

		d.NotificationsChannel <- notificationEvent
	}
	return nil
}

func (d *Daemon) CloseNotificationAsUser(id uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.Notifications[id]; exists {

		d.conn.Emit(dbus.ObjectPath("/org/freedesktop/Notifications"), "org.freedesktop.Notifications.NotificationClosed", id, 2)
		slog.Debug(strings.Join([]string{"User closed notification ", strconv.Itoa(int(id))}, ""))

		notificationEvent := NotificationEvent{
			Notification: d.Notifications[id],
			Created:      false,
			Modified:     false,
			Deleted:      true,
		}
		delete(d.Notifications, id)

		d.NotificationsChannel <- notificationEvent
	}
	return nil
}
