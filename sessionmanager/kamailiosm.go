/*
Real-time Charging System for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

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
	"errors"
	"fmt"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/kamevapi"
	"log/syslog"
	"regexp"
	"time"
)

func NewKamailioSessionManager(smKamCfg *config.SmKamConfig, rater, cdrsrv engine.Connector, cdrDb engine.CdrStorage) (*KamailioSessionManager, error) {
	ksm := &KamailioSessionManager{cfg: smKamCfg, rater: rater, cdrsrv: cdrsrv, cdrDb: cdrDb, conns: make(map[string]*kamevapi.KamEvapi)}
	return ksm, nil
}

type KamailioSessionManager struct {
	cfg      *config.SmKamConfig
	rater    engine.Connector
	cdrsrv   engine.Connector
	cdrDb    engine.CdrStorage
	conns    map[string]*kamevapi.KamEvapi
	sessions []*Session
}

func (self *KamailioSessionManager) onCgrAuth(evData []byte, connId string) {
	kev, err := NewKamEvent(evData)
	if err != nil {
		engine.Logger.Info(fmt.Sprintf("<SM-Kamailio> ERROR unmarshalling event: %s, error: %s", evData, err.Error()))
	}
	if kev.GetReqType(utils.META_DEFAULT) == utils.META_NONE { // Do not process this request
		return
	}
	if kev.MissingParameter() {
		if kar, err := kev.AsKamAuthReply(0.0, errors.New(utils.ERR_MANDATORY_IE_MISSING)); err != nil {
			engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Failed building auth reply %s", err.Error()))
		} else if err = self.conns[connId].Send(kar.String()); err != nil {
			engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Failed sending auth reply %s", err.Error()))
		}
		return
	}
	var remainingDuration float64
	if err = self.rater.GetDerivedMaxSessionTime(*kev.AsStoredCdr(), &remainingDuration); err != nil {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Could not get max session time for %s, error: %s", kev.GetUUID(), err.Error()))
	}
	if kar, err := kev.AsKamAuthReply(remainingDuration, err); err != nil {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Failed building auth reply %s", err.Error()))
	} else if err = self.conns[connId].Send(kar.String()); err != nil {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Failed sending auth reply %s", err.Error()))
	}
}

func (self *KamailioSessionManager) onCallStart(evData []byte, connId string) {
	kamEv, err := NewKamEvent(evData)
	if err != nil {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> ERROR unmarshalling event: %s, error: %s", evData, err.Error()))
	}
	if kamEv.GetReqType(utils.META_DEFAULT) == utils.META_NONE { // Do not process this request
		return
	}
	if kamEv.MissingParameter() {
		self.DisconnectSession(kamEv, "", utils.ERR_MANDATORY_IE_MISSING)
		return
	}
	s := NewSession(kamEv, "", self)
	if s != nil {
		self.sessions = append(self.sessions, s)
	}
}

func (self *KamailioSessionManager) onCallEnd(evData []byte, connId string) {
	kev, err := NewKamEvent(evData)
	if err != nil {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> ERROR unmarshalling event: %s, error: %s", evData, err.Error()))
	}
	if kev.GetReqType(utils.META_DEFAULT) == utils.META_NONE { // Do not process this request
		return
	}
	if kev.MissingParameter() {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Mandatory IE missing out of event: %+v", kev))
	}
	go self.ProcessCdr(kev.AsStoredCdr())
	s := self.GetSession(kev.GetUUID())
	if s == nil { // Not handled by us
		return
	}
	self.RemoveSession(s.eventStart.GetUUID()) // Unreference it early so we avoid concurrency
	if err := s.Close(kev); err != nil {       // Stop loop, refund advanced charges and save the costs deducted so far to database
		engine.Logger.Err(err.Error())
	}
}

func (self *KamailioSessionManager) Connect() error {
	var err error
	eventHandlers := map[*regexp.Regexp][]func([]byte, string){
		regexp.MustCompile("CGR_AUTH_REQUEST"): []func([]byte, string){self.onCgrAuth},
		regexp.MustCompile("CGR_CALL_START"):   []func([]byte, string){self.onCallStart},
		regexp.MustCompile("CGR_CALL_END"):     []func([]byte, string){self.onCallEnd},
	}
	errChan := make(chan error)
	for _, connCfg := range self.cfg.Connections {
		connId := utils.GenUUID()
		if self.conns[connId], err = kamevapi.NewKamEvapi(connCfg.EvapiAddr, connId, connCfg.Reconnects, eventHandlers, engine.Logger.(*syslog.Writer)); err != nil {
			return err
		}
		go func() { // Start reading in own goroutine, return on error
			if err := self.conns[connId].ReadEvents(); err != nil {
				errChan <- err
			}
		}()
	}
	err = <-errChan // Will keep the Connect locked until the first error in one of the connections
	return err
}

func (self *KamailioSessionManager) DisconnectSession(ev engine.Event, connId, notify string) error {
	sessionIds := ev.GetSessionIds()
	disconnectEv := &KamSessionDisconnect{Event: CGR_SESSION_DISCONNECT, HashEntry: sessionIds[0], HashId: sessionIds[1], Reason: notify}
	if err := self.conns[connId].Send(disconnectEv.String()); err != nil {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Failed sending disconnect request, error %s, connection id: %s", err.Error(), connId))
		return err
	}
	return nil
}
func (self *KamailioSessionManager) RemoveSession(uuid string) {
	for i, ss := range self.sessions {
		if ss.eventStart.GetUUID() == uuid {
			self.sessions = append(self.sessions[:i], self.sessions[i+1:]...)
			return
		}
	}
}

// Searches and return the session with the specifed uuid
func (self *KamailioSessionManager) GetSession(uuid string) *Session {
	for _, s := range self.sessions {
		if s.eventStart.GetUUID() == uuid {
			return s
		}
	}
	return nil
}
func (self *KamailioSessionManager) DebitInterval() time.Duration {
	return self.cfg.DebitInterval
}
func (self *KamailioSessionManager) CdrDb() engine.CdrStorage {
	return self.cdrDb
}
func (self *KamailioSessionManager) Rater() engine.Connector {
	return self.rater
}

func (self *KamailioSessionManager) ProcessCdr(cdr *engine.StoredCdr) error {
	if self.cdrsrv == nil {
		return nil
	}
	var reply string
	if err := self.cdrsrv.ProcessCdr(cdr, &reply); err != nil {
		engine.Logger.Err(fmt.Sprintf("<SM-Kamailio> Failed processing CDR, cgrid: %s, accid: %s, error: <%s>", cdr.CgrId, cdr.AccId, err.Error()))
	}
	return nil
}

func (sm *KamailioSessionManager) WarnSessionMinDuration(sessionUuid, connId string) {
}
func (self *KamailioSessionManager) Shutdown() error {
	return nil
}
