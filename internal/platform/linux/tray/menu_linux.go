//go:build linux

package tray

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	menuPath  = dbus.ObjectPath("/MenuBar")
	menuIface = "com.canonical.dbusmenu"

	// The menu never changes after it is exported.
	menuRevision = 1
)

type menuLayout struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

type menuItemProperties struct {
	ID         int32
	Properties map[string]dbus.Variant
}

type menuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

// dbusMenu serves a flat menu over com.canonical.dbusmenu. Item IDs are the
// 1-based positions in items; ID 0 is the root.
type dbusMenu struct {
	items []MenuItem
}

func exportMenu(conn *dbus.Conn, items []MenuItem) error {
	if err := conn.Export(&dbusMenu{items: items}, menuPath, menuIface); err != nil {
		return err
	}
	_, err := prop.Export(conn, menuPath, prop.Map{menuIface: {
		"Version":       {Value: uint32(3)},
		"TextDirection": {Value: "ltr"},
		"Status":        {Value: "normal"},
		"IconThemePath": {Value: []string{}},
	}})
	if err != nil {
		return err
	}
	return conn.Export(introspect.Introspectable(menuIntrospection), menuPath, "org.freedesktop.DBus.Introspectable")
}

func (m *dbusMenu) properties(id int32) (map[string]dbus.Variant, bool) {
	if id == 0 {
		return map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}, true
	}
	if id < 0 || int(id) > len(m.items) {
		return nil, false
	}
	item := m.items[id-1]
	if item.Separator {
		return map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}, true
	}
	return map[string]dbus.Variant{
		"label":   dbus.MakeVariant(item.Label),
		"enabled": dbus.MakeVariant(true),
		"visible": dbus.MakeVariant(true),
	}, true
}

func filterProperties(props map[string]dbus.Variant, names []string) map[string]dbus.Variant {
	if len(names) == 0 {
		return props
	}
	out := make(map[string]dbus.Variant, len(names))
	for _, name := range names {
		if v, ok := props[name]; ok {
			out[name] = v
		}
	}
	return out
}

func (m *dbusMenu) layout(id, depth int32, names []string) menuLayout {
	props, _ := m.properties(id)
	l := menuLayout{ID: id, Properties: filterProperties(props, names), Children: []dbus.Variant{}}
	if id == 0 && depth != 0 {
		for i := range m.items {
			l.Children = append(l.Children, dbus.MakeVariant(m.layout(int32(i+1), depth-1, names)))
		}
	}
	return l
}

func (m *dbusMenu) GetLayout(parentID, recursionDepth int32, propertyNames []string) (uint32, menuLayout, *dbus.Error) {
	if _, ok := m.properties(parentID); !ok {
		return 0, menuLayout{}, unknownItem(parentID)
	}
	return menuRevision, m.layout(parentID, recursionDepth, propertyNames), nil
}

func (m *dbusMenu) GetGroupProperties(ids []int32, propertyNames []string) ([]menuItemProperties, *dbus.Error) {
	if len(ids) == 0 {
		for i := range m.items {
			ids = append(ids, int32(i+1))
		}
	}
	out := []menuItemProperties{}
	for _, id := range ids {
		if props, ok := m.properties(id); ok {
			out = append(out, menuItemProperties{ID: id, Properties: filterProperties(props, propertyNames)})
		}
	}
	return out, nil
}

func (m *dbusMenu) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	props, ok := m.properties(id)
	if !ok {
		return dbus.Variant{}, unknownItem(id)
	}
	v, ok := props[name]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs",
			[]interface{}{fmt.Sprintf("menu item %d has no property %q", id, name)})
	}
	return v, nil
}

func (m *dbusMenu) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if _, ok := m.properties(id); !ok {
		return unknownItem(id)
	}
	if eventID == "clicked" && id > 0 {
		if fn := m.items[id-1].OnClick; fn != nil {
			go fn()
		}
	}
	return nil
}

func (m *dbusMenu) EventGroup(events []menuEvent) ([]int32, *dbus.Error) {
	idErrors := []int32{}
	for _, e := range events {
		if err := m.Event(e.ID, e.EventID, e.Data, e.Timestamp); err != nil {
			idErrors = append(idErrors, e.ID)
		}
	}
	return idErrors, nil
}

func (m *dbusMenu) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

func (m *dbusMenu) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
}

func unknownItem(id int32) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs",
		[]interface{}{fmt.Sprintf("unknown menu item %d", id)})
}

const menuIntrospection = `<node>
	<interface name="com.canonical.dbusmenu">
		<property name="Version" type="u" access="read"/>
		<property name="TextDirection" type="s" access="read"/>
		<property name="Status" type="s" access="read"/>
		<property name="IconThemePath" type="as" access="read"/>
		<method name="GetLayout">
			<arg name="parentId" type="i" direction="in"/>
			<arg name="recursionDepth" type="i" direction="in"/>
			<arg name="propertyNames" type="as" direction="in"/>
			<arg name="revision" type="u" direction="out"/>
			<arg name="layout" type="(ia{sv}av)" direction="out"/>
		</method>
		<method name="GetGroupProperties">
			<arg name="ids" type="ai" direction="in"/>
			<arg name="propertyNames" type="as" direction="in"/>
			<arg name="properties" type="a(ia{sv})" direction="out"/>
		</method>
		<method name="GetProperty">
			<arg name="id" type="i" direction="in"/>
			<arg name="name" type="s" direction="in"/>
			<arg name="value" type="v" direction="out"/>
		</method>
		<method name="Event">
			<arg name="id" type="i" direction="in"/>
			<arg name="eventId" type="s" direction="in"/>
			<arg name="data" type="v" direction="in"/>
			<arg name="timestamp" type="u" direction="in"/>
		</method>
		<method name="EventGroup">
			<arg name="events" type="a(isvu)" direction="in"/>
			<arg name="idErrors" type="ai" direction="out"/>
		</method>
		<method name="AboutToShow">
			<arg name="id" type="i" direction="in"/>
			<arg name="needUpdate" type="b" direction="out"/>
		</method>
		<method name="AboutToShowGroup">
			<arg name="ids" type="ai" direction="in"/>
			<arg name="updatesNeeded" type="ai" direction="out"/>
			<arg name="idErrors" type="ai" direction="out"/>
		</method>
		<signal name="ItemsPropertiesUpdated">
			<arg name="updatedProps" type="a(ia{sv})"/>
			<arg name="removedProps" type="a(ias)"/>
		</signal>
		<signal name="LayoutUpdated">
			<arg name="revision" type="u"/>
			<arg name="parent" type="i"/>
		</signal>
		<signal name="ItemActivationRequested">
			<arg name="id" type="i"/>
			<arg name="timestamp" type="u"/>
		</signal>
	</interface>` + prop.IntrospectDataString + introspect.IntrospectDataString + `</node>`
