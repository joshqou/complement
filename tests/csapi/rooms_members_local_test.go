package csapi_tests

import (
	"testing"

	"github.com/matrix-org/complement/internal/b"
	"github.com/matrix-org/complement/internal/client"
	"github.com/matrix-org/complement/runtime"
)

func TestMembersLocal(t *testing.T) {
	deployment := Deploy(t, b.BlueprintAlice)
	defer deployment.Destroy(t)

	alice := deployment.Client(t, "hs1", "@alice:hs1")
	// Here we don't use the BlueprintOneToOneRoom because else Bob would be able to see Alice's presence changes through
	// that pre-existing one-on-one DM room. So we exclude that here.
	bob := deployment.RegisterUser(t, "hs1", "bob", "bobspassword", false)
	roomID := alice.CreateRoom(t, map[string]interface{}{"preset": "public_chat"})

	bob.MustDoFunc(
		t, "PUT", []string{"_matrix", "client", "v3", "presence", bob.UserID, "status"},
		client.WithJSONBody(t, map[string]interface{}{
			"presence": "online",
		}),
	)

	_, incrementalSyncTokenBeforeBobJoinsRoom := alice.MustSync(t, client.SyncReq{TimeoutMillis: "0"})
	bob.JoinRoom(t, roomID, []string{})

	t.Run("Parallel", func(t *testing.T) {
		// sytest: New room members see their own join event
		t.Run("New room members see their own join event", func(t *testing.T) {
			t.Parallel()
			// SyncJoinedTo already checks everything we need to know
			alice.MustSyncUntil(t, client.SyncReq{}, client.SyncJoinedTo(alice.UserID, roomID))
		})

		// sytest: Existing members see new members' join events
		t.Run("Existing members see new members' join events", func(t *testing.T) {
			t.Parallel()
			// SyncJoinedTo already checks everything we need to know
			alice.MustSyncUntil(t, client.SyncReq{Since: incrementalSyncTokenBeforeBobJoinsRoom}, client.SyncJoinedTo(bob.UserID, roomID))
		})

		// sytest: Existing members see new members' presence
		// Split into initial and incremental sync cases in Complement.
		t.Run("Existing members see new members' presence (in initial sync)", func(t *testing.T) {
			runtime.SkipIf(t, runtime.Dendrite) // FIXME: https://github.com/matrix-org/matrix-spec/issues/1374
			t.Parallel()
			// First we sync to make sure bob to have joined the room...
			alice.MustSyncUntil(t, client.SyncReq{}, client.SyncJoinedTo(bob.UserID, roomID))
			// ...and then we do another initial sync - this time waiting for bob's presence - to confirm we can get that.
			alice.MustSyncUntil(t, client.SyncReq{},
				client.SyncJoinedTo(bob.UserID, roomID),
				client.SyncPresenceHas(bob.UserID, nil),
			)
		})

		// sytest: Existing members see new members' presence
		// Split into initial and incremental sync cases in Complement.
		t.Run("Existing members see new members' presence (in incremental sync)", func(t *testing.T) {
			t.Parallel()
			alice.MustSyncUntil(t, client.SyncReq{Since: incrementalSyncTokenBeforeBobJoinsRoom},
				client.SyncJoinedTo(bob.UserID, roomID),
				client.SyncPresenceHas(bob.UserID, nil),
			)
		})
	})

}
