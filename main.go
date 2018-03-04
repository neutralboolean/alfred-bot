//Alfred bot
//alfred(main).go
package main 

import (
	//"alfred-bot/config"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
	"syscall"
	"time"
	"os"
	"os/signal"
)


var m *Meeting

func main() {
	session, err := discordgo.New("Bot " + Token)
	fmt.Println("Initialized...")

	if err != nil {
		fmt.Println("Error while creating Discord session: ", err)
		return
	}

	//add messageCreate handler
	session.AddHandler(messageCreate)
	fmt.Println("Added handler...")

	err = session.Open()
	if err != nil {
		fmt.Println("Error while opening Discord session", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV, os.Interrupt, os.Kill)
	<-sc

	session.Close()
	fmt.Println("Closing...")
}

func messageCreate(dses *discordgo.Session, dmsg *discordgo.MessageCreate) {
	if dmsg.Author.ID == dses.State.User.ID {
		return
	}

	chanID := dmsg.ChannelID
	// var m Meeting

	if strings.HasPrefix(dmsg.Content, "!make meeting") {
		m = MakeMeeting("Meeting", dmsg, dses)
		fmt.Println("Made meeting: " + m.GetMeetingTitle())
		dses.ChannelMessageSend(chanID, fmt.Sprintf("I have booked a meeting for %v", m.Setting))
	} else if strings.HasPrefix(dmsg.Content, "!rsvp") {
		s := make([]string, 1)
		s[0] = dmsg.Author.ID
		err := m.RSVP(s, dses)
		if err == nil {
			fmt.Println("RSVP'd someone for meeting: " + m.GetMeetingTitle())
			dses.ChannelMessageSend(chanID, fmt.Sprintf("I have %v marked down as attending %v.", dmsg.Author.Username, m.GetMeetingTitle()))
		} else {
			fmt.Println("Someone RSVP'd for a bad meeting...")
			dses.ChannelMessageSend(chanID, "I do not have records of any such meeting.")
		}
	} else if strings.HasPrefix(dmsg.Content, "!start meeting") {
		if CanAttend(m, dmsg, dses) {
			var atmentions string = ""
			for _,a := range m.Attendants {
				atmentions = atmentions + a.Mention()
			}
			
			dses.ChannelMessageSend(chanID, fmt.Sprintf("Pardon me but %v is set to begin now. Attendees: %v", m.GetMeetingTitle(), atmentions))
			m.StartMeeting(dses)
			fmt.Println("Started meeting: " + m.GetMeetingTitle())
		} else {
			if m != nil {
				fmt.Println("Someone with insufficient privilege tried to start meeting <" + m.GetMeetingTitle() + ">...")
				dses.ChannelMessageSend(chanID, "Apologies but I don't believe you are a part of that meeting. Please \"!rsvp\" in advance.")
			} else {
				fmt.Println("Tried to start meeting that doesn't exist")
				dses.ChannelMessageSend(chanID, "I do not have notes about any such meeting. Very sorry.")
				dses.ChannelMessageSend(chanID, "However, you may book a meeting at this moment.")
			}
		}
	} else if strings.HasPrefix(dmsg.Content, "!clean meeting") {
		if CanAttend(m, dmsg, dses) {
			var err0 error

			dses.ChannelMessageSend(chanID, "I shall take care of it right away.")
			m, err0 = m.CleanMeeting(dses)
			if err0 != nil {
				fmt.Println("Failed to properly clear meeting: " + m.GetMeetingTitle())
			} else {
				fmt.Println("Cleaned up after meeting...")
			}
		} else {
			fmt.Println("Someone with insufficient privileges tried to clean up a meeting...")
			dses.ChannelMessageSend(chanID, "I shall await the attendees' decision to end the meeting.")
		}
	} else if strings.HasPrefix(dmsg.Content, "!you can go") {
		var sett string

		//Only the server master should be able to dismiss Alfred. Else you'll have trolls kicking him off all day.
			//might increase range to support those with Admin privileges in future
		if (dmsg.Author.ID == Master) {
			l, _:= time.LoadLocation("America/New_York")
			t := time.Now().In(l)
			if t.Hour() > Nite {
				sett = "night"
			} else {
				sett = "day"
			}

			dses.ChannelMessageSend(chanID, fmt.Sprintf("Very well then. Good %s, Master %s", sett, dmsg.Author.Username))
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		} else {
			dses.ChannelMessageSend(chanID, "A very good joke, sir.")
		}
	} else if strings.HasPrefix(dmsg.Content, "?commands") {
		dses.ChannelMessageSend(chanID, fmt.Sprintf("Commands must be at the beginning of the message::\n\t!make meeting : makes a meeting\n\t!start meeting : start a previously made meeting\n\t!clean meeting : cleans up after a made meeting"))
	} else {
		for _, user := range dmsg.Mentions {
			if user.ID == dses.State.User.ID {
				fmt.Println("Message-mention log: " + dmsg.ContentWithMentionsReplaced())

				if dmsg.Author.ID == Master {
					dses.ChannelMessageSend(chanID, fmt.Sprintf("Yes, Master %s?", dmsg.Author.Username))
				} else { 
					var sett string
					l, _:= time.LoadLocation("America/New_York")
					t := time.Now().In(l)
					if t.Hour() < Noon {
						sett = "day"
					} else if t.Hour() > Noon && t.Hour() < Nite {
						sett = "afternoon"
					} else if t.Hour() > Nite {
						sett = "evening"
					}

					dses.ChannelMessageSend(chanID, fmt.Sprintf("Good %v, %v.", sett, dmsg.Author.Username))
				}
			}
		}
	}
}

func CanAttend(meeting *Meeting, dmsg *discordgo.MessageCreate, dses *discordgo.Session) bool {
	var channel *discordgo.Channel
	var member *discordgo.Member
	var err error

	if m == nil  {
		fmt.Println("Meeting was nil...")
		return false
	} else if dmsg == nil {
		fmt.Println("MessageCreate was nil...")
		return false
	} else if dses == nil {
		fmt.Println("Session was nil...")
		return false
	}

	channel, err = dses.Channel(dmsg.ChannelID)
	if err != nil {
		fmt.Println("Failed to get Channel struct")
		return false
	}

	member, err = dses.GuildMember(channel.GuildID, dmsg.Author.ID)
	if err != nil {
		fmt.Println("Failed to get Member struct")
		return false
	}

	for _, role := range member.Roles {
		fmt.Println(role + " vs " + m.AttendRole.ID)
		if role == m.AttendRole.ID {
			return true
		}
	}

	return false
}
