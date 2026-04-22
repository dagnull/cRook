function nameForm() {
  return {
    name: '',
    async setName() {
      if (!this.name.trim()) return;
      await fetch('/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: this.name.trim() }),
      });
      window.location.reload();
    },
  };
}

function lobby() {
  return {};
}

const SUIT_SYMBOL = { yellow: '🟡', red: '🔴', green: '🟢', black: '⚫' };

function gameRoom(roomID, playerID, playerName) {
  return {
    roomID,
    playerID,
    playerName,

    // Connection
    ws: null,
    connected: false,

    // State
    phase: 'waiting',
    players: [],       // PlayerView[]
    yourHand: [],      // Card[]
    trump: null,
    currentBid: 0,
    bidderID: '',
    bidTurn: '',
    currentTrick: { plays: [] },
    nestSize: 5,
    events: [],        // recent event log

    // UI inputs
    bidAmount: 70,
    selectedCard: null,
    selectedDiscards: [],

    connect() {
      const proto = location.protocol === 'https:' ? 'wss' : 'ws';
      this.ws = new WebSocket(`${proto}://${location.host}/rooms/${roomID}/ws`);

      this.ws.onopen = () => { this.connected = true; };
      this.ws.onclose = () => {
        this.connected = false;
        setTimeout(() => this.connect(), 2000);
      };
      this.ws.onmessage = (e) => this.handleMessage(JSON.parse(e.data));
    },

    send(obj) {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify(obj));
      }
    },

    handleMessage(msg) {
      this.log(msg);
      switch (msg.kind) {
        case 'state_sync':       this.applyStateSync(msg.payload); break;
        case 'player_joined':    this.onPlayerJoined(msg.payload); break;
        case 'player_left':      this.onPlayerLeft(msg.payload); break;
        case 'card_dealt':       this.onCardDealt(msg.payload); break;
        case 'bid_placed':       this.onBidPlaced(msg.payload); break;
        case 'player_passed':    this.onPassed(msg.payload); break;
        case 'bidding_won':      this.onBiddingWon(msg.payload); break;
        case 'trump_named':      this.onTrumpNamed(msg.payload); break;
        case 'nest_set':         this.phase = 'playing'; break;
        case 'card_played':      this.onCardPlayed(msg.payload); break;
        case 'trick_won':        this.onTrickWon(msg.payload); break;
        case 'round_scored':     this.onRoundScored(msg.payload); break;
        case 'error':            alert('Error: ' + msg.payload.message); break;
      }
    },

    applyStateSync(s) {
      this.phase        = s.phase;
      this.players      = s.players || [];
      this.yourHand     = s.your_hand || [];
      this.trump        = (s.trump && s.trump !== 'none') ? s.trump : null;
      this.currentBid   = s.current_bid;
      this.bidderID     = s.bidder_id;
      this.bidTurn      = s.bid_turn;
      this.currentTrick = s.current_trick || { plays: [] };
      this.nestSize     = s.nest_size;
      if (this.bidAmount < this.currentBid + 5) {
        this.bidAmount = this.currentBid + 5 || 70;
      }
    },

    onPlayerJoined(p) {
      if (!this.players.find(x => x.id === p.player_id)) {
        this.players.push({ id: p.player_id, name: p.name, team: -1, card_count: 0, score: 0 });
      }
    },

    onPlayerLeft(p) {
      this.players = this.players.filter(x => x.id !== p.player_id);
    },

    onCardDealt(p) {
      if (p.player_id === this.playerID && p.cards.length > 0) {
        this.yourHand = p.cards;
        this.phase = 'bidding';
      }
      const pl = this.players.find(x => x.id === p.player_id);
      if (pl) pl.card_count = p.cards.length > 0 ? p.cards.length : pl.card_count || 13;
    },

    onBidPlaced(p) {
      this.currentBid = p.amount;
      this.bidderID = p.player_id;
      this.advanceBidTurn();
      this.bidAmount = this.currentBid + 5;
    },

    onPassed(p) {
      const pl = this.players.find(x => x.id === p.player_id);
      if (pl) pl.passed = true;
      this.advanceBidTurn();
    },

    advanceBidTurn() {
      // Server is authoritative — we just wait for next state_sync or bid event.
      // For optimistic UI we skip passed players.
      const active = this.players.filter(p => !p.passed);
      const idx = active.findIndex(p => p.id === this.bidTurn);
      if (active.length > 0) {
        this.bidTurn = active[(idx + 1) % active.length].id;
      }
    },

    onBiddingWon(p) {
      this.bidderID = p.player_id;
      this.currentBid = p.amount;
      this.phase = 'nesting';
    },

    onTrumpNamed(p) {
      this.trump = p.trump;
      // state_sync arrives automatically after dispatch and will update yourHand with nest cards.
    },

    onCardPlayed(p) {
      if (!this.currentTrick.plays) this.currentTrick = { plays: [] };
      this.currentTrick.plays.push(p);
      if (p.player_id === this.playerID) {
        this.yourHand = this.yourHand.filter(c => !(c.suit === p.card.suit && c.value === p.card.value));
      }
      const pl = this.players.find(x => x.id === p.player_id);
      if (pl) pl.card_count = Math.max(0, (pl.card_count || 1) - 1);
    },

    onTrickWon(p) {
      setTimeout(() => { this.currentTrick = { plays: [] }; }, 1500);
    },

    onRoundScored(p) {
      this.phase = 'scoring';
      for (const score of p.scores) {
        const pl = this.players.find(x => x.id === score.player_id);
        if (pl) pl.score = (pl.score || 0) + score.points;
      }
    },

    // ── Actions ────────────────────────────────────────────────────────────────

    startGame() { this.send({ kind: 'start_game' }); },

    placeBid() {
      this.send({ kind: 'bid', amount: parseInt(this.bidAmount) });
    },

    pass() { this.send({ kind: 'pass' }); },

    nameTrump(suit) {
      this.send({ kind: 'name_trump', trump: suit });
    },

    toggleDiscard(card) {
      const key = card.suit + ':' + card.value;
      const idx = this.selectedDiscards.findIndex(c => c.suit === card.suit && c.value === card.value);
      if (idx >= 0) {
        this.selectedDiscards.splice(idx, 1);
      } else if (this.selectedDiscards.length < this.nestSize) {
        this.selectedDiscards.push(card);
      }
    },

    isDiscardSelected(card) {
      return this.selectedDiscards.some(c => c.suit === card.suit && c.value === card.value);
    },

    setNest() {
      if (this.selectedDiscards.length !== this.nestSize) {
        alert(`Select exactly ${this.nestSize} cards to discard.`);
        return;
      }
      this.send({ kind: 'set_nest', discards: this.selectedDiscards });
      this.selectedDiscards = [];
    },

    playCard(card) {
      this.send({ kind: 'play_card', card });
    },

    // ── Computed helpers ───────────────────────────────────────────────────────

    isMyBidTurn() { return this.bidTurn === this.playerID; },
    isMyPlayTurn() {
      if (!this.currentTrick) return false;
      const plays = this.currentTrick.plays || [];
      const leader = this.players.findIndex(p => p.id === this.trickLeaderID);
      const expected = (leader + plays.length) % this.players.length;
      return expected >= 0 && this.players[expected]?.id === this.playerID;
    },
    isBidder() { return this.bidderID === this.playerID; },

    suitName(suit) {
      return ['yellow', 'red', 'green', 'black'][suit] || 'none';
    },

    cardLabel(card) {
      if (!card) return '?';
      if (card.value === 0) return '🐦 Rook';
      return `${card.value} ${SUIT_SYMBOL[card.suit] || card.suit}`;
    },

    playerName(id) {
      return this.players.find(p => p.id === id)?.name || id;
    },

    log(msg) {
      this.events.unshift(msg);
      if (this.events.length > 30) this.events.pop();
    },
  };
}