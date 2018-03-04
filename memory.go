//memory.go
//Holds anything [states] that Alfred might need to remember, even across sessions.
package main

import (
	"github.com/bwmarrin/discordgo"
)

var Meetings []*Meeting
var WaitingForName, WaitingForTime bool	//follow up

// var RoleCreated bool = false
// var SavedRole Role discordgo.Role

type UsersList struct {
	Users []discordgo.User
}

var Timezones map[string]UsersList
//e.g. Timezones["est"] = append(Timezones["est"], new-user)