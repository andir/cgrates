/*
Rating system designed to be used in VoIP Carriers World
Copyright (C) 2012  Radu Ioan Fericean

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package sessionmanager

import (
	"github.com/rif/cgrates/timespans"
	"log"
	"time"
)

const (
	DEBIT_PERIOD = 10 * time.Second
)

var (
	// sample storage for the provided direct implementation
	storageGetter, _ = timespans.NewRedisStorage("tcp:127.0.0.1:6379", 10)
)

// Interface for the session delegate objects
type SessionDelegate interface {
	// Called on freeswitch's hearbeat event
	OnHeartBeat(*Event)
	// Called on freeswitch's answer event
	OnChannelAnswer(*Event, *Session)
	// Called on freeswitch's hangup event
	OnChannelHangupComplete(*Event, *Session)
	// The method to be called inside the debit loop
	LoopAction(*Session, *timespans.CallDescriptor)
	// Returns a storage getter for the sesssion to use
	GetDebitPeriod() time.Duration
}

// Sample SessionDelegate calling the timespans methods directly
type DirectSessionDelegate byte

func (dsd *DirectSessionDelegate) OnHeartBeat(ev *Event) {
	log.Print("direct hearbeat")
}

func (dsd *DirectSessionDelegate) OnChannelAnswer(ev *Event, s *Session) {
	log.Print("direct answer")
}

func (dsd *DirectSessionDelegate) OnChannelHangupComplete(ev *Event, s *Session) {
	log.Print("direct hangup")
}

func (dsd *DirectSessionDelegate) LoopAction(s *Session, cd *timespans.CallDescriptor) {
	cc, err := cd.Debit()
	if err != nil {
		log.Printf("Could not complete debit opperation: %v", err)
	}
	s.CallCosts = append(s.CallCosts, cc)
	cd.Amount = DEBIT_PERIOD.Seconds()
	if remainingSeconds, err := cd.GetMaxSessionTime(); remainingSeconds < DEBIT_PERIOD.Seconds() || err != nil {
		log.Print("Not enough money for another debit period!")
	}
}

func (dsd *DirectSessionDelegate) GetDebitPeriod() time.Duration {
	return DEBIT_PERIOD
}

// Sample SessionDelegate calling the timespans methods through the RPC interface
type RPCSessionDelegate byte

func (rsd *RPCSessionDelegate) OnHeartBeat(ev *Event) {
	log.Print("rpc hearbeat")
}

func (rsd *RPCSessionDelegate) OnChannelAnswer(ev *Event, s *Session) {
	log.Print("rpc answer")
}

func (rsd *RPCSessionDelegate) OnChannelHangupComplete(ev *Event, s *Session) {
	log.Print("rpc hangup")
}

func (rsd *RPCSessionDelegate) LoopAction(s *Session, cd *timespans.CallDescriptor) {
	log.Print("Rpc debit")
}

func (rsd *RPCSessionDelegate) GetDebitPeriod() time.Duration {
	return DEBIT_PERIOD
}