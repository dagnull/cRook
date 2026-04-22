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
    players: [],         // PlayerView[] — ordered by seat
    yourHand: [],        // Card[]
    trump: null,
    currentBid: 0,
    bidderID: '',
    bidTurn: '',         // player ID whose turn it is to bid
    trickLeaderID: '',   // player ID who leads the current trick
    currentTrick: { plays: [] },
    nestSize: 5,
    events: [],          // recent event log

    // UI inputs
    bidAmount: 70,
    selectedCard: null,
    selectedDiscards: [],
    pendingAction: false, // true while waiting for server to ack last action

    // Toasts
    toasts: [],
    _toastSeq: 0,

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

    sendAction(obj) {
      if (this.pendingAction) return;
      this.pendingAction = true;
      this.send(obj);
    },

    handleMessage(msg) {
      this.log(msg);
      switch (msg.kind) {
        case 'state_sync':    this.applyStateSync(msg.payload); break;
        case 'player_joined': this.onPlayerJoined(msg.payload); break;
        case 'player_left':   this.onPlayerLeft(msg.payload); break;
        case 'card_dealt':    this.onCardDealt(msg.payload); break;
        case 'game_started':  this.toast('Cards dealt — bidding begins!', 'info'); break;
        case 'bid_placed':    this.onBidPlaced(msg.payload); break;
        case 'player_passed': this.onPassed(msg.payload); break;
        case 'bidding_won':   this.onBiddingWon(msg.payload); break;
        case 'trump_named':   this.onTrumpNamed(msg.payload); break;
        case 'nest_set':      this.onNestSet(); break;
        case 'card_played':   this.onCardPlayed(msg.payload); break;
        case 'trick_won':     this.onTrickWon(msg.payload); break;
        case 'round_scored':  this.onRoundScored(msg.payload); break;
        case 'error':         this.toast(msg.payload.message, 'error', 5000); break;
      }
    },

    applyStateSync(s) {
      this.pendingAction = false;
      this.phase         = s.phase;
      this.players       = s.players || [];
      this.yourHand      = s.your_hand || [];
      this.trump         = (s.trump && s.trump !== 'none') ? s.trump : null;
      this.currentBid    = s.current_bid;
      this.bidderID      = s.bidder_id;
      this.bidTurn       = s.bid_turn;
      this.trickLeaderID = s.trick_leader_id || '';
      this.currentTrick  = s.current_trick || { plays: [] };
      this.nestSize      = s.nest_size;
      if (this.bidAmount < this.currentBid + 5) {
        this.bidAmount = this.currentBid + 5 || 70;
      }
    },

    onPlayerJoined(p) {
      if (!this.players.find(x => x.id === p.player_id)) {
        this.players.push({ id: p.player_id, name: p.name, team: -1, card_count: 0, score: 0 });
      }
      if (p.player_id !== this.playerID) {
        this.toast(`${p.name} joined`, 'info', 2500);
      }
    },

    onPlayerLeft(p) {
      this.players = this.players.filter(x => x.id !== p.player_id);
      this.toast(`${p.name} left`, 'warning', 3000);
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
      this.bidAmount = this.currentBid + 5;
      const name = p.player_id === this.playerID ? 'You' : this.playerName(p.player_id);
      this.toast(`${name} bid ${p.amount}`, 'info', 2500);
    },

    onPassed(p) {
      const name = p.player_id === this.playerID ? 'You' : this.playerName(p.player_id);
      this.toast(`${name} passed`, 'warning', 2500);
    },

    onBiddingWon(p) {
      this.bidderID = p.player_id;
      this.currentBid = p.amount;
      this.phase = 'nesting';
      const name = p.player_id === this.playerID ? 'You' : this.playerName(p.player_id);
      this.toast(`${name} won the bid at ${p.amount}!`, 'success', 4000);
    },

    onTrumpNamed(p) {
      this.trump = p.trump;
      const suit = p.trump ? p.trump.charAt(0).toUpperCase() + p.trump.slice(1) : '';
      this.toast(`Trump is ${suit} ${SUIT_SYMBOL[p.trump] || ''}`, 'success', 3500);
    },

    onNestSet() {
      this.toast('Nest set — play begins!', 'info', 3000);
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
      const name = p.player_id === this.playerID ? 'You' : this.playerName(p.player_id);
      const pts = p.points > 0 ? ` (${p.points} pts)` : '';
      this.toast(`${name} won the trick${pts}`, 'success', 3000);
      setTimeout(() => { this.currentTrick = { plays: [] }; }, 1500);
    },

    onRoundScored(p) {
      this.phase = 'scoring';
      for (const score of p.scores) {
        const pl = this.players.find(x => x.id === score.player_id);
        if (pl) pl.score = (pl.score || 0) + score.points;
      }
      if (p.bidder_met) {
        const name = p.bidder_id === this.playerID ? 'You' : this.playerName(p.bidder_id);
        this.toast(`Round over! ${name} made the bid of ${p.bid_amount}.`, 'success', 5000);
      } else {
        const name = p.bidder_id === this.playerID ? 'You were' : `${this.playerName(p.bidder_id)} was`;
        this.toast(`Round over! ${name} set (bid ${p.bid_amount}).`, 'warning', 5000);
      }
    },

    // ── Toast system ──────────────────────────────────────────────────────────

    toast(message, type = 'info', duration = 3500) {
      const id = ++this._toastSeq;
      this.toasts.push({ id, message, type, fading: false });

      // Begin fade-out slightly before removal so animation completes.
      setTimeout(() => {
        const t = this.toasts.find(t => t.id === id);
        if (t) t.fading = true;
      }, duration);

      setTimeout(() => {
        this.toasts = this.toasts.filter(t => t.id !== id);
      }, duration + 320);
    },

    // ── Actions ───────────────────────────────────────────────────────────────

    startGame() { this.send({ kind: 'start_game' }); },

    placeBid() {
      this.sendAction({ kind: 'bid', amount: parseInt(this.bidAmount) });
    },

    pass() { this.sendAction({ kind: 'pass' }); },

    nameTrump(suit) {
      this.sendAction({ kind: 'name_trump', trump: suit });
    },

    toggleDiscard(card) {
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
        this.toast(`Select exactly ${this.nestSize} cards to discard.`, 'error');
        return;
      }
      this.sendAction({ kind: 'set_nest', discards: this.selectedDiscards });
      this.selectedDiscards = [];
    },

    playCard(card) {
      this.sendAction({ kind: 'play_card', card });
    },

    // ── Turn helpers ──────────────────────────────────────────────────────────

    currentTurnID() {
      if (this.phase === 'bidding') return this.bidTurn;
      if (this.phase === 'playing') {
        const plays = this.currentTrick.plays || [];
        const leaderIdx = this.players.findIndex(p => p.id === this.trickLeaderID);
        if (leaderIdx < 0 || this.players.length === 0) return '';
        return this.players[(leaderIdx + plays.length) % this.players.length]?.id || '';
      }
      if (this.phase === 'nesting') return this.bidderID;
      return '';
    },

    isMyTurn()     { return this.currentTurnID() === this.playerID; },
    isMyBidTurn()  { return this.phase === 'bidding' && this.bidTurn === this.playerID; },
    isMyPlayTurn() { return this.phase === 'playing' && this.currentTurnID() === this.playerID; },
    isBidder()     { return this.bidderID === this.playerID; },

    trickPlayOrder() {
      if (!this.trickLeaderID || this.players.length === 0) return this.players;
      const leaderIdx = this.players.findIndex(p => p.id === this.trickLeaderID);
      if (leaderIdx < 0) return this.players;
      return Array.from({ length: this.players.length }, (_, i) =>
        this.players[(leaderIdx + i) % this.players.length]
      );
    },

    cardPlayedBy(playerID) {
      return (this.currentTrick.plays || []).find(p => p.player_id === playerID)?.card || null;
    },

    cardLabel(card) {
      if (!card) return '?';
      if (card.value === 0) return '🐦 Rook';
      return `${card.value} ${SUIT_SYMBOL[card.suit] || card.suit}`;
    },

    playerName(id) {
      return this.players.find(p => p.id === id)?.name || id;
    },

    ordinal(n) {
      if (n === 2) return 'nd';
      if (n === 3) return 'rd';
      return 'th';
    },

    log(msg) {
      this.events.unshift(msg);
      if (this.events.length > 30) this.events.pop();
    },
  };
}