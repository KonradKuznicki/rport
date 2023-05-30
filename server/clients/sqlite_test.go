package clients

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientsSqliteProvider(t *testing.T) {
	ctx := context.Background()
	keepLost := hour
	p := NewFakeClientProvider(t, &keepLost)
	defer p.Close()
	noObsoleteProvider := newSqliteProvider(p.db, nil)

	// verify add clients
	c1 := New(t).Logger(testLog).Build()                                                   // active
	c2 := New(t).DisconnectedDuration(5 * time.Minute).Logger(testLog).Build()             // disconnected
	c3 := New(t).DisconnectedDuration(keepLost - time.Millisecond).Logger(testLog).Build() // disconnected
	c4 := New(t).DisconnectedDuration(keepLost).Logger(testLog).Build()                    // disconnected
	c5 := New(t).DisconnectedDuration(keepLost + time.Millisecond).Logger(testLog).Build() // obsolete
	require.NoError(t, p.Save(ctx, c1))
	require.NoError(t, p.Save(ctx, c2))
	require.NoError(t, p.Save(ctx, c3))
	require.NoError(t, p.Save(ctx, c4))
	require.NoError(t, p.Save(ctx, c5))

	// verify Get clients
	gotAll, err := p.GetNonObsoleteClients(ctx, testLog)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{c1, c2, c3, c4}, gotAll)

	// verify no obsolete Get clients
	gotAll, err = noObsoleteProvider.GetNonObsoleteClients(ctx, testLog)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{c1, c2, c3, c4, c5}, gotAll)

	// verify delete obsolete clients
	gotObsolete, err := p.Get(ctx, c5.GetID(), testLog)
	require.NoError(t, err)
	require.EqualValues(t, c5, gotObsolete)

	require.NoError(t, p.DeleteObsolete(ctx, testLog))
	gotObsolete, err = p.Get(ctx, c5.GetID(), testLog)
	require.NoError(t, err)
	require.Nil(t, gotObsolete)

	gotAll, err = p.GetNonObsoleteClients(ctx, testLog)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{c1, c2, c3, c4}, gotAll)

	// verify not found
	gotNone, err := p.Get(ctx, "unknown-id", testLog)
	require.NoError(t, err)
	require.Nil(t, gotNone)

	// verify update
	d := time.Date(2020, 11, 5, 12, 11, 20, 0, time.UTC)
	c1.DisconnectedAt = &d
	require.NoError(t, p.Save(ctx, c1))
	gotUpdated, err := p.Get(ctx, c1.GetID(), testLog)
	require.NoError(t, err)
	require.EqualValues(t, c1, gotUpdated)
	gotAll, err = p.GetNonObsoleteClients(ctx, testLog)
	require.NoError(t, err)
	assert.ElementsMatch(t, []*Client{c1, c2, c3, c4}, gotAll)
}
