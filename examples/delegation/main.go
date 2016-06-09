// Copyright (c) 2014 - Max Ekman <max@looplab.se>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package delegation contains a simple runnable example of a CQRS/ES app.
package main

import (
	"fmt"
	"log"

	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/messaging/local"
	"github.com/looplab/eventhorizon/storage/memory"
)

func main() {
	// Create the event bus that distributes events.
	eventBus := local.NewEventBus()
	eventBus.AddGlobalHandler(&LoggerSubscriber{})

	// Create the event store.
	eventStore := memory.NewEventStore(eventBus)

	// Create the aggregate repository.
	repository, err := eventhorizon.NewCallbackRepository(eventStore)
	if err != nil {
		log.Fatalf("could not create repository: %s", err)
	}

	// Register an aggregate factory.
	repository.RegisterAggregate(&InvitationAggregate{},
		func(id eventhorizon.UUID) eventhorizon.Aggregate {
			return &InvitationAggregate{
				AggregateBase: eventhorizon.NewAggregateBase(id),
			}
		},
	)

	// Create the aggregate command handler.
	handler, err := eventhorizon.NewAggregateCommandHandler(repository)
	if err != nil {
		log.Fatalf("could not create command handler: %s", err)
	}

	// Register the domain aggregates with the dispather. Remember to check for
	// errors here in a real app!
	handler.SetAggregate(&InvitationAggregate{}, &CreateInvite{})
	handler.SetAggregate(&InvitationAggregate{}, &AcceptInvite{})
	handler.SetAggregate(&InvitationAggregate{}, &DeclineInvite{})

	// Create the command bus and register the handler for the commands.
	commandBus := local.NewCommandBus()
	commandBus.SetHandler(handler, &CreateInvite{})
	commandBus.SetHandler(handler, &AcceptInvite{})
	commandBus.SetHandler(handler, &DeclineInvite{})

	// Create and register a read model for individual invitations.
	invitationRepository := memory.NewReadRepository()
	invitationProjector := NewInvitationProjector(invitationRepository)
	eventBus.AddHandler(invitationProjector, &InviteCreated{})
	eventBus.AddHandler(invitationProjector, &InviteAccepted{})
	eventBus.AddHandler(invitationProjector, &InviteDeclined{})

	// Create and register a read model for a guest list.
	eventID := eventhorizon.NewUUID()
	guestListRepository := memory.NewReadRepository()
	guestListProjector := NewGuestListProjector(guestListRepository, eventID)
	eventBus.AddHandler(guestListProjector, &InviteCreated{})
	eventBus.AddHandler(guestListProjector, &InviteAccepted{})
	eventBus.AddHandler(guestListProjector, &InviteDeclined{})

	// Issue some invitations and responses.
	// Note that Athena tries to decline the event, but that is not allowed
	// by the domain logic in InvitationAggregate. The result is that she is
	// still accepted.
	athenaID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&CreateInvite{InvitationID: athenaID, Name: "Athena", Age: 42})
	commandBus.HandleCommand(&AcceptInvite{InvitationID: athenaID})
	err = commandBus.HandleCommand(&DeclineInvite{InvitationID: athenaID})
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}

	hadesID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&CreateInvite{InvitationID: hadesID, Name: "Hades"})
	commandBus.HandleCommand(&AcceptInvite{InvitationID: hadesID})

	zeusID := eventhorizon.NewUUID()
	commandBus.HandleCommand(&CreateInvite{InvitationID: zeusID, Name: "Zeus"})
	commandBus.HandleCommand(&DeclineInvite{InvitationID: zeusID})

	// Read all invites.
	invitations, _ := invitationRepository.FindAll()
	for _, i := range invitations {
		fmt.Printf("invitation: %#v\n", i)
	}

	// Read the guest list.
	guestList, _ := guestListRepository.Find(eventID)
	fmt.Printf("guest list: %#v\n", guestList)
}

// LoggerSubscriber is a simple event handler for logging all events.
type LoggerSubscriber struct{}

// HandleEvent implements the HandleEvent method of the EventHandler interface.
func (l *LoggerSubscriber) HandleEvent(event eventhorizon.Event) {
	log.Printf("event: %#v\n", event)
}
