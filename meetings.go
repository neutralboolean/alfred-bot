//meetings.go
package main 

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	// "http"
	"time"
)

const gold int = 15844367

type Meeting struct {
	Title string
	Discriminator int
	GuildID string
	Channel *discordgo.Channel
	Setting time.Time
	Attendants []discordgo.User
	AttendRole *discordgo.Role
}

func MakeMeeting(title string, dmsg *discordgo.MessageCreate, dses *discordgo.Session)(*Meeting) {
	discriminator := 0
	for _, meet := range Meetings {
		if meet.Title == title {
			discriminator++
		}
	}

	tempchan,_ := dses.State.Channel(dmsg.ChannelID)
	newrole, err := dses.GuildRoleCreate(tempchan.GuildID)
	if err == nil {
		newrole, err = dses.GuildRoleEdit(tempchan.GuildID, newrole.ID, fmt.Sprintf("%v-%d Attendee", title, discriminator), gold, false, 0, true)
		if err == nil {
			err = dses.State.RoleAdd(tempchan.GuildID, newrole)
			if err == nil {
				//RoleCreated = true
			} else {
				fmt.Println("Failed to add new Role <" + newrole.Name + "> to Session.State.")
			}
		} else {
			fmt.Println("Failed to edit new Role.")
		}
	} else {
		fmt.Println("Failed to create new Role.")
	}

	m := Meeting{Title: title, Discriminator: discriminator, Channel: nil, GuildID: tempchan.GuildID, Setting: time.Now(), AttendRole: newrole}
	Meetings = append(Meetings, &m)
	err = m.InitRSVP(dmsg.Author.ID, dses)

	return &m
}

func (m *Meeting) InitRSVP(userid string, dses *discordgo.Session) error {
	m.Attendants = make([]discordgo.User, 1)
	u, err := dses.User(userid)
	if err == nil {
		m.Attendants[0] = *u
	} else {
		fmt.Println("Failed to get User struct")
	}

	err = dses.GuildMemberRoleAdd(m.GuildID, userid, m.AttendRole.ID)
	if err != nil {
		fmt.Println("Failed to properly add role to user " + userid + "...")
		return err
	}

	return nil
}

func (m *Meeting) RSVP(userids []string, dses *discordgo.Session) error {
	if m == nil {
		return errors.New("Tried to RSVP for non-extant meeting.")
	}

	for _, userid := range userids {
		u, err := dses.User(userid)
		if err == nil {
			for _, a := range m.Attendants {
				if userid == a.ID {
					fmt.Println("User tried to RSVP again for same meeting")
					return RSVPError{"User tried to RSVP again for same meeting"}
				}
			}
			m.Attendants[0] = *u
		} else {
			fmt.Println("Failed to get User struct")
		}

		err = dses.GuildMemberRoleAdd(m.GuildID, userid, m.AttendRole.ID)
		if err != nil {
			fmt.Println("Failed to properly add role to user " + userid +"...")
			return err
		}
	}

	return nil
}

//mentions Users with the correct roles and opens a temporary Channel for mentioned members to be moved to
func (m *Meeting) StartMeeting(dses *discordgo.Session) {
	if m == nil {
		fmt.Println("No meeting to start...")
		return
	}

	newchan, err := dses.GuildChannelCreate(m.GuildID, m.GetMeetingTitle(), "voice")
	if err != nil {
		fmt.Println("Failed to correctly create new channel...")
	} else {
		m.Channel = newchan
		for _, a := range m.Attendants {
			err = dses.GuildMemberMove(m.GuildID, a.ID, m.Channel.ID)
			if err != nil {
				fmt.Println("Failed to move user <" + a.ID +"> to channel <" + m.Channel.ID + ">...")
			}
		}
	}
}

func (m *Meeting) GetMeetingTitle() string {
	if m == nil {
		panic("Tried to get Title string of a Meeting that doesn't exist")
	}

	if m.Discriminator == 0 {
		return m.Title
	} else {
		return fmt.Sprintf("%v #%v", m.Title, m.Discriminator)
	}
}

//cleans up roles (i.e. removes temp roles and closes opened Channels, etc)
func (m *Meeting) CleanMeeting(dses *discordgo.Session) (*Meeting, error) {
	if m == nil {
		fmt.Println("Tried to clean up after non-extant meeting.")
		return nil, nil
	}

	var i int = 0
	for _, meeting := range Meetings {
		if m.Title == meeting.Title {
			if m.Discriminator == meeting.Discriminator {
				break
			}
		}
		i++
	}

	if i < len(Meetings)-1 {
		i++
		Meetings[i] = Meetings[len(Meetings)-1]
		Meetings[len(Meetings)-1] = nil
		Meetings = Meetings[:len(Meetings)-1]
	}
	
	var err error
	if m.Channel != nil {
		m.Channel, err = dses.ChannelDelete(m.Channel.ID)
		if err != nil {
			fmt.Printf("Failed to properly remove channel; error code: %v\n", err)
		}
		fmt.Println("Proper removed channel...")
	}
	
	return nil, dses.GuildRoleDelete(m.GuildID, m.AttendRole.ID)
}