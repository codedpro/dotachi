/* ============================================================
   Dotachi Admin Panel - Vanilla JS
   ============================================================ */

(function () {
  'use strict';

  // ---- State ----

  let token = localStorage.getItem('dotachi_token') || '';
  let apiBase = localStorage.getItem('dotachi_api_url') || 'http://localhost:8080';
  let currentUser = null;
  let nodesCache = [];
  let usersCache = [];
  let refreshInterval = null;
  let monitorInterval = null;
  let assignRoomId = null;
  let shardModalUserId = null;
  let shardModalAction = null; // 'add' or 'remove'

  // ---- DOM Refs ----

  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => document.querySelectorAll(sel);

  // ---- Toast ----

  function toast(message, type) {
    type = type || 'error';
    const container = $('#toast-container');
    const el = document.createElement('div');
    el.className = 'toast toast-' + type;
    el.textContent = message;
    container.appendChild(el);
    setTimeout(function () {
      el.style.opacity = '0';
      el.style.transition = 'opacity 0.3s';
      setTimeout(function () { el.remove(); }, 300);
    }, 4000);
  }

  // ---- API ----

  function api(method, path, body) {
    const url = apiBase.replace(/\/+$/, '') + path;
    const opts = {
      method: method,
      headers: { 'Content-Type': 'application/json' }
    };
    if (token) {
      opts.headers['Authorization'] = 'Bearer ' + token;
    }
    if (body) {
      opts.body = JSON.stringify(body);
    }
    return fetch(url, opts).then(function (res) {
      return res.json().then(function (data) {
        if (!res.ok) {
          var msg = data.error || data.message || ('HTTP ' + res.status);
          throw new Error(msg);
        }
        return data;
      });
    });
  }

  // ---- Helpers ----

  function show(el) {
    if (typeof el === 'string') el = $(el);
    if (el) el.classList.remove('hidden');
  }

  function hide(el) {
    if (typeof el === 'string') el = $(el);
    if (el) el.classList.add('hidden');
  }

  function setLoading(btn, loading) {
    var text = btn.querySelector('.btn-text');
    var spinner = btn.querySelector('.spinner');
    if (loading) {
      btn.disabled = true;
      if (text) text.style.display = 'none';
      if (spinner) show(spinner);
    } else {
      btn.disabled = false;
      if (text) text.style.display = '';
      if (spinner) hide(spinner);
    }
  }

  function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  function formatDate(dateStr) {
    if (!dateStr) return '--';
    try {
      var d = new Date(dateStr);
      return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch (e) {
      return dateStr;
    }
  }

  function formatShards(amount) {
    if (amount == null || amount === '' || isNaN(amount)) return '--';
    var num = Number(amount);
    return num.toLocaleString('en-US');
  }

  function formatExpiry(dateStr) {
    if (!dateStr) return { html: '<span class="expiry-text expiry-ok">--</span>', urgent: false };

    var now = new Date();
    var exp = new Date(dateStr);
    var diff = exp.getTime() - now.getTime();

    if (diff <= 0) {
      return { html: '<span class="expiry-text expiry-expired">Expired</span>', urgent: false };
    }

    var totalSeconds = Math.floor(diff / 1000);
    var days = Math.floor(totalSeconds / 86400);
    var hours = Math.floor((totalSeconds % 86400) / 3600);
    var minutes = Math.floor((totalSeconds % 3600) / 60);

    var text = '';
    if (days > 0) {
      text = days + 'd ' + hours + 'h left';
    } else if (hours > 0) {
      text = hours + 'h ' + minutes + 'm left';
    } else {
      text = minutes + 'm left';
    }

    var isUrgent = diff < 86400000; // less than 1 day
    var cls = isUrgent ? 'expiry-urgent' : 'expiry-ok';

    return {
      html: '<span class="expiry-text ' + cls + '">' + escapeHtml(text) + '</span>',
      urgent: isUrgent
    };
  }

  // ---- Screen switching ----

  function showLogin() {
    show('#login-screen');
    hide('#dashboard-screen');
    clearIntervals();
  }

  function showDashboard() {
    hide('#login-screen');
    show('#dashboard-screen');
    $('#api-url-input').value = apiBase;
    if (currentUser) {
      $('#user-display').textContent = currentUser.display_name || currentUser.phone;
    }
    loadAllData();
    startAutoRefresh();
  }

  function clearIntervals() {
    if (refreshInterval) { clearInterval(refreshInterval); refreshInterval = null; }
    if (monitorInterval) { clearInterval(monitorInterval); monitorInterval = null; }
  }

  function startAutoRefresh() {
    clearIntervals();
    refreshInterval = setInterval(loadAllData, 30000);
    monitorInterval = setInterval(loadMonitor, 10000);
  }

  // ---- Auth ----

  function handleLogin(e) {
    e.preventDefault();
    var btn = e.target.querySelector('button[type="submit"]');
    var phone = $('#login-phone').value.trim();
    var pw = $('#login-password').value;
    var urlVal = $('#login-api-url').value.trim();

    if (urlVal) {
      apiBase = urlVal;
      localStorage.setItem('dotachi_api_url', apiBase);
    }

    setLoading(btn, true);
    api('POST', '/auth/login', { phone: phone, password: pw })
      .then(function (data) {
        token = data.token;
        currentUser = data.user;
        localStorage.setItem('dotachi_token', token);
        if (!currentUser.is_admin) {
          toast('Access denied: admin only', 'error');
          logout();
          return;
        }
        showDashboard();
      })
      .catch(function (err) {
        toast(err.message);
      })
      .finally(function () {
        setLoading(btn, false);
      });
  }

  function checkToken() {
    if (!token) { showLogin(); return; }
    api('GET', '/auth/me')
      .then(function (user) {
        if (!user.is_admin) {
          toast('Access denied: admin only', 'error');
          logout();
          return;
        }
        currentUser = user;
        showDashboard();
      })
      .catch(function () {
        showLogin();
      });
  }

  function logout() {
    token = '';
    currentUser = null;
    localStorage.removeItem('dotachi_token');
    showLogin();
  }

  // ---- Tabs ----

  function switchTab(tabName) {
    $$('.tab').forEach(function (t) { t.classList.toggle('active', t.dataset.tab === tabName); });
    $$('.tab-panel').forEach(function (p) { p.classList.toggle('active', p.id === 'tab-' + tabName); });

    if (tabName === 'monitor') {
      loadMonitor();
    }
  }

  // ---- Load all data ----

  function loadAllData() {
    loadNodes();
    loadRooms();
    loadUsers();
    loadMonitor();
  }

  // ---- Nodes ----

  function loadNodes() {
    show('#nodes-loading');
    hide('#nodes-empty');
    api('GET', '/nodes')
      .then(function (data) {
        nodesCache = data.nodes || [];
        renderNodes();
        populateNodeDropdowns();
      })
      .catch(function (err) {
        toast('Failed to load nodes: ' + err.message);
      })
      .finally(function () {
        hide('#nodes-loading');
      });
  }

  function renderNodes() {
    var tbody = $('#nodes-tbody');
    tbody.innerHTML = '';

    if (nodesCache.length === 0) {
      show('#nodes-empty');
      return;
    }
    hide('#nodes-empty');

    nodesCache.forEach(function (node) {
      var tr = document.createElement('tr');
      var statusBadge = node.is_active
        ? '<span class="badge badge-muted" id="node-status-' + node.id + '">Unknown</span>'
        : '<span class="badge badge-danger">Inactive</span>';

      tr.innerHTML =
        '<td class="mono">' + node.id + '</td>' +
        '<td>' + escapeHtml(node.name) + '</td>' +
        '<td class="mono">' + escapeHtml(node.host) + '</td>' +
        '<td class="mono">' + node.api_port + '</td>' +
        '<td>' + statusBadge + '</td>' +
        '<td>' + (node.room_count || 0) + '</td>' +
        '<td><button class="btn-icon ping-btn" data-id="' + node.id + '">Ping</button></td>';
      tbody.appendChild(tr);
    });
  }

  function pingNode(id) {
    var badge = $('#node-status-' + id);
    if (badge) {
      badge.className = 'badge badge-warning';
      badge.textContent = 'Pinging...';
    }
    api('POST', '/nodes/' + id + '/ping')
      .then(function (data) {
        if (!badge) return;
        if (data.status === 'ok') {
          badge.className = 'badge badge-success';
          badge.textContent = 'Healthy';
        } else {
          badge.className = 'badge badge-danger';
          badge.textContent = 'Down';
        }
      })
      .catch(function () {
        if (badge) {
          badge.className = 'badge badge-danger';
          badge.textContent = 'Error';
        }
      });
  }

  function addNode() {
    var name = $('#node-name').value.trim();
    var host = $('#node-host').value.trim();
    var port = parseInt($('#node-port').value) || 7443;
    var secret = $('#node-secret').value.trim();

    if (!name || !host || !secret) {
      toast('Name, host, and API secret are required');
      return;
    }

    var btn = $('#add-node-submit');
    btn.disabled = true;

    api('POST', '/nodes', { name: name, host: host, api_port: port, api_secret: secret })
      .then(function () {
        toast('Node added successfully', 'success');
        hide('#add-node-form');
        $('#node-name').value = '';
        $('#node-host').value = '';
        $('#node-port').value = '7443';
        $('#node-secret').value = '';
        loadNodes();
      })
      .catch(function (err) {
        toast('Failed to add node: ' + err.message);
      })
      .finally(function () {
        btn.disabled = false;
      });
  }

  function populateNodeDropdowns() {
    var sel1 = $('#room-node');
    var sel2 = $('#rooms-node-filter');

    sel1.innerHTML = '<option value="">Select node...</option>';
    // Keep first option in filter
    var firstOpt = sel2.querySelector('option[value=""]');
    sel2.innerHTML = '';
    sel2.appendChild(firstOpt);

    nodesCache.forEach(function (node) {
      var opt1 = document.createElement('option');
      opt1.value = node.id;
      opt1.textContent = node.name + ' (' + node.host + ')';
      sel1.appendChild(opt1);

      var opt2 = document.createElement('option');
      opt2.value = node.id;
      opt2.textContent = node.name;
      sel2.appendChild(opt2);
    });
  }

  // ---- Rooms ----

  function buildRoomQuery() {
    var params = [];
    var q = $('#rooms-search').value.trim();
    var nodeId = $('#rooms-node-filter').value;
    var priv = $('#rooms-status-filter').value;

    if (q) params.push('q=' + encodeURIComponent(q));
    if (nodeId) params.push('node_id=' + nodeId);
    if (priv) params.push('is_private=' + priv);
    params.push('per_page=100');

    return '/rooms?' + params.join('&');
  }

  function loadRooms() {
    show('#rooms-loading');
    hide('#rooms-empty');
    api('GET', buildRoomQuery())
      .then(function (data) {
        renderRooms(data.rooms || []);
      })
      .catch(function (err) {
        toast('Failed to load rooms: ' + err.message);
      })
      .finally(function () {
        hide('#rooms-loading');
      });
  }

  function renderRooms(rooms) {
    var tbody = $('#rooms-tbody');
    tbody.innerHTML = '';

    if (rooms.length === 0) {
      show('#rooms-empty');
      return;
    }
    hide('#rooms-empty');

    rooms.forEach(function (room) {
      var tr = document.createElement('tr');
      var ownerText = room.owner_display_name
        ? escapeHtml(room.owner_display_name) + ' <span class="mono">(#' + (room.owner_id || '') + ')</span>'
        : '<span class="mono" style="color:var(--text-muted)">Unassigned</span>';

      var visBadge = room.is_private
        ? '<span class="badge badge-warning">Private</span>'
        : '<span class="badge badge-success">Public</span>';

      var statusBadge = room.is_active
        ? '<span class="badge badge-success">Active</span>'
        : '<span class="badge badge-danger">Inactive</span>';

      // Expiry countdown
      var expiry = formatExpiry(room.expires_at);

      // Type badges (shared + hourly cost)
      var typeBadges = '<div class="room-type-badges">';
      if (room.is_shared) {
        typeBadges += '<span class="badge badge-shard">Shared</span>';
        if (room.hourly_cost != null && room.hourly_cost > 0) {
          typeBadges += '<div class="hourly-cost-label">' + formatShards(room.hourly_cost) + '/hr</div>';
        }
      } else {
        typeBadges += '<span class="badge badge-muted">Standard</span>';
      }
      typeBadges += '</div>';

      tr.innerHTML =
        '<td class="mono">' + room.id + '</td>' +
        '<td>' + escapeHtml(room.name) + '</td>' +
        '<td class="mono">' + escapeHtml(room.hub_name) + '</td>' +
        '<td>' + escapeHtml(room.node_name) + '</td>' +
        '<td>' + ownerText + '</td>' +
        '<td>' + room.current_players + '/' + room.max_players + '</td>' +
        '<td>' + expiry.html + '</td>' +
        '<td>' + typeBadges + '</td>' +
        '<td>' + visBadge + '</td>' +
        '<td>' + statusBadge + '</td>' +
        '<td><button class="btn-icon assign-btn" data-id="' + room.id + '" data-name="' + escapeHtml(room.name) + '">Assign Owner</button></td>';
      tbody.appendChild(tr);
    });
  }

  function createRoom() {
    var name = $('#room-name').value.trim();
    var nodeId = parseInt($('#room-node').value);
    var maxPlayers = parseInt($('#room-max').value);
    var isPrivate = $('#room-private').checked;
    var password = $('#room-password').value;
    var gameTag = $('#room-game-tag').value.trim();
    var expiresAt = $('#room-expires-at').value;
    var isShared = $('#room-shared').checked;
    var hourlyCost = parseInt($('#room-hourly-cost').value);

    if (!name) { toast('Room name is required'); return; }
    if (!nodeId) { toast('Select a node'); return; }

    var body = {
      name: name,
      node_id: nodeId,
      max_players: maxPlayers,
      is_private: isPrivate
    };
    if (isPrivate && password) {
      body.password = password;
    }
    if (gameTag) {
      body.game_tag = gameTag;
    }
    if (expiresAt) {
      body.expires_at = new Date(expiresAt).toISOString();
    }
    if (isShared) {
      body.is_shared = true;
      if (hourlyCost && hourlyCost > 0) {
        body.hourly_cost = hourlyCost;
      }
    }

    var btn = $('#create-room-submit');
    btn.disabled = true;

    api('POST', '/admin/rooms', body)
      .then(function () {
        toast('Room created successfully', 'success');
        hide('#create-room-form');
        $('#room-name').value = '';
        $('#room-private').checked = false;
        $('#room-password').value = '';
        $('#room-password-field').style.display = 'none';
        $('#room-game-tag').value = '';
        $('#room-expires-at').value = '';
        $('#room-shared').checked = false;
        $('#room-hourly-cost').value = '';
        $('#room-hourly-cost-field').style.display = 'none';
        loadRooms();
      })
      .catch(function (err) {
        toast('Failed to create room: ' + err.message);
      })
      .finally(function () {
        btn.disabled = false;
      });
  }

  // ---- Assign Owner Modal ----

  function openAssignModal(roomId, roomName) {
    assignRoomId = roomId;
    $('#assign-room-name').textContent = roomName;
    $('#assign-search').value = '';
    renderAssignUsers('');
    show('#assign-modal');
  }

  function closeAssignModal() {
    hide('#assign-modal');
    assignRoomId = null;
  }

  function renderAssignUsers(query) {
    var tbody = $('#assign-users-tbody');
    tbody.innerHTML = '';

    var q = query.toLowerCase();
    var filtered = usersCache.filter(function (u) {
      if (!q) return true;
      return (u.phone && u.phone.toLowerCase().indexOf(q) !== -1) ||
             (u.display_name && u.display_name.toLowerCase().indexOf(q) !== -1);
    });

    filtered.forEach(function (u) {
      var tr = document.createElement('tr');
      tr.innerHTML =
        '<td class="mono">' + u.id + '</td>' +
        '<td class="mono">' + escapeHtml(u.phone) + '</td>' +
        '<td>' + escapeHtml(u.display_name) + '</td>' +
        '<td><button class="btn-icon do-assign-btn" data-uid="' + u.id + '">Assign</button></td>';
      tbody.appendChild(tr);
    });
  }

  function doAssign(userId) {
    if (!assignRoomId) return;
    api('POST', '/admin/rooms/' + assignRoomId + '/assign-owner', { user_id: parseInt(userId) })
      .then(function () {
        toast('Owner assigned successfully', 'success');
        closeAssignModal();
        loadRooms();
      })
      .catch(function (err) {
        toast('Failed to assign owner: ' + err.message);
      });
  }

  // ---- Shard Modal ----

  function openShardModal(userId, userName, action) {
    shardModalUserId = userId;
    shardModalAction = action;
    $('#shard-modal-title').textContent = action === 'add' ? 'Add Shards' : 'Remove Shards';
    $('#shard-modal-user').textContent = userName || ('User #' + userId);
    $('#shard-amount').value = '';
    $('#shard-description').value = '';
    show('#shard-modal');
    $('#shard-amount').focus();
  }

  function closeShardModal() {
    hide('#shard-modal');
    shardModalUserId = null;
    shardModalAction = null;
  }

  function submitShardModal() {
    if (!shardModalUserId || !shardModalAction) return;

    var amount = parseInt($('#shard-amount').value);
    var description = $('#shard-description').value.trim();

    if (!amount || amount <= 0) {
      toast('Enter a valid shard amount');
      return;
    }

    var endpoint = '/admin/users/' + shardModalUserId + '/' + shardModalAction + '-shards';
    var body = { amount: amount };
    if (description) {
      body.description = description;
    }

    var btn = $('#shard-modal-submit');
    btn.disabled = true;

    api('POST', endpoint, body)
      .then(function () {
        var verb = shardModalAction === 'add' ? 'added to' : 'removed from';
        toast(formatShards(amount) + ' shards ' + verb + ' user', 'success');
        closeShardModal();
        loadUsers();
      })
      .catch(function (err) {
        toast('Failed to ' + shardModalAction + ' shards: ' + err.message);
      })
      .finally(function () {
        btn.disabled = false;
      });
  }

  // ---- Users ----

  function loadUsers() {
    show('#users-loading');
    hide('#users-empty');
    api('GET', '/admin/users')
      .then(function (data) {
        usersCache = data.users || [];
        renderUsers();
      })
      .catch(function (err) {
        toast('Failed to load users: ' + err.message);
      })
      .finally(function () {
        hide('#users-loading');
      });
  }

  function renderUsers() {
    var tbody = $('#users-tbody');
    tbody.innerHTML = '';

    var q = ($('#users-search').value || '').trim().toLowerCase();
    var filtered = usersCache.filter(function (u) {
      if (!q) return true;
      return (u.phone && u.phone.toLowerCase().indexOf(q) !== -1) ||
             (u.display_name && u.display_name.toLowerCase().indexOf(q) !== -1);
    });

    if (filtered.length === 0) {
      show('#users-empty');
      return;
    }
    hide('#users-empty');

    filtered.forEach(function (u) {
      var tr = document.createElement('tr');
      var roleBadge = u.is_admin
        ? '<span class="badge badge-success">Admin</span>'
        : '<span class="badge badge-muted">User</span>';

      var shardDisplay = '<span class="shard-amount">' + formatShards(u.shard_balance) + '</span>';

      var displayName = escapeHtml(u.display_name) || escapeHtml(u.phone);

      tr.innerHTML =
        '<td class="mono">' + u.id + '</td>' +
        '<td class="mono">' + escapeHtml(u.phone) + '</td>' +
        '<td>' + escapeHtml(u.display_name) + '</td>' +
        '<td>' + shardDisplay + '</td>' +
        '<td>' + roleBadge + '</td>' +
        '<td>' + formatDate(u.created_at) + '</td>' +
        '<td>' +
          '<div class="user-actions">' +
            '<button class="btn-icon btn-shard add-shards-btn" data-uid="' + u.id + '" data-uname="' + escapeHtml(displayName) + '">+ Shards</button>' +
            '<button class="btn-icon btn-shard-remove remove-shards-btn" data-uid="' + u.id + '" data-uname="' + escapeHtml(displayName) + '">- Shards</button>' +
          '</div>' +
        '</td>';
      tbody.appendChild(tr);
    });
  }

  // ---- Live Monitor ----

  function loadMonitor() {
    // Only fetch if the monitor tab is active
    if (!$('#tab-monitor').classList.contains('active')) return;

    show('#monitor-loading');
    hide('#monitor-empty');

    var fetchList = [
      api('GET', '/rooms?per_page=100'),
      api('GET', '/nodes'),
      api('GET', '/admin/users')
    ];

    // Try to fetch shard stats; the endpoint may not exist yet
    var shardStatsPromise = api('GET', '/admin/shard-stats').catch(function () {
      return null;
    });

    Promise.all([
      Promise.all(fetchList),
      shardStatsPromise
    ])
      .then(function (results) {
        var coreResults = results[0];
        var shardStats = results[1];

        var rooms = coreResults[0].rooms || [];
        var nodes = coreResults[1].nodes || [];
        var users = coreResults[2].users || [];

        var activeRooms = rooms.filter(function (r) { return r.is_active; });
        var totalPlayers = activeRooms.reduce(function (sum, r) { return sum + r.current_players; }, 0);

        $('#stat-active-rooms').textContent = activeRooms.length;
        $('#stat-total-players').textContent = totalPlayers;
        $('#stat-nodes-online').textContent = nodes.length;

        // Calculate total shards in circulation from user balances
        var totalShards = users.reduce(function (sum, u) {
          return sum + (Number(u.shard_balance) || 0);
        }, 0);
        $('#stat-total-shards').textContent = formatShards(totalShards);

        // Daily shard revenue from dedicated endpoint, or fallback
        if (shardStats && shardStats.daily_revenue != null) {
          $('#stat-daily-revenue').textContent = formatShards(shardStats.daily_revenue);
        } else if (shardStats && shardStats.daily_shard_revenue != null) {
          $('#stat-daily-revenue').textContent = formatShards(shardStats.daily_shard_revenue);
        } else {
          // If no endpoint available, show the sum from users as a fallback indicator
          $('#stat-daily-revenue').textContent = '--';
        }

        renderMonitor(activeRooms);
      })
      .catch(function (err) {
        toast('Monitor error: ' + err.message);
      })
      .finally(function () {
        hide('#monitor-loading');
      });
  }

  function renderMonitor(rooms) {
    var tbody = $('#monitor-tbody');
    tbody.innerHTML = '';

    if (rooms.length === 0) {
      show('#monitor-empty');
      return;
    }
    hide('#monitor-empty');

    rooms.forEach(function (room) {
      var tr = document.createElement('tr');
      var pct = room.max_players > 0 ? Math.round((room.current_players / room.max_players) * 100) : 0;
      var capClass = pct < 50 ? 'low' : (pct < 80 ? 'mid' : 'high');

      tr.innerHTML =
        '<td>' + escapeHtml(room.name) + '</td>' +
        '<td>' + escapeHtml(room.node_name) + '</td>' +
        '<td>' + room.current_players + '</td>' +
        '<td>' + room.max_players + '</td>' +
        '<td>' +
          pct + '% ' +
          '<div class="capacity-bar"><div class="capacity-fill ' + capClass + '" style="width:' + pct + '%"></div></div>' +
        '</td>' +
        '<td><span class="badge badge-success">Active</span></td>';
      tbody.appendChild(tr);
    });
  }

  // ---- Event Binding ----

  function bindEvents() {
    // Login
    $('#login-form').addEventListener('submit', handleLogin);
    $('#login-api-url').value = apiBase;

    // Logout
    $('#logout-btn').addEventListener('click', logout);

    // API URL change
    $('#api-url-input').addEventListener('change', function () {
      apiBase = this.value.trim() || apiBase;
      localStorage.setItem('dotachi_api_url', apiBase);
      toast('API URL updated', 'info');
      loadAllData();
    });

    // Tabs
    $$('.tab').forEach(function (t) {
      t.addEventListener('click', function () {
        switchTab(this.dataset.tab);
      });
    });

    // Nodes: toggle form
    $('#add-node-toggle').addEventListener('click', function () {
      var form = $('#add-node-form');
      form.classList.contains('hidden') ? show(form) : hide(form);
    });
    $('#add-node-cancel').addEventListener('click', function () { hide('#add-node-form'); });
    $('#add-node-submit').addEventListener('click', addNode);

    // Nodes: ping delegation
    $('#nodes-tbody').addEventListener('click', function (e) {
      var btn = e.target.closest('.ping-btn');
      if (btn) pingNode(btn.dataset.id);
    });

    // Rooms: toggle form
    $('#create-room-toggle').addEventListener('click', function () {
      var form = $('#create-room-form');
      form.classList.contains('hidden') ? show(form) : hide(form);
    });
    $('#create-room-cancel').addEventListener('click', function () { hide('#create-room-form'); });
    $('#create-room-submit').addEventListener('click', createRoom);

    // Room: slider label
    $('#room-max').addEventListener('input', function () {
      $('#room-max-val').textContent = this.value;
    });

    // Room: private toggle
    $('#room-private').addEventListener('change', function () {
      $('#room-password-field').style.display = this.checked ? '' : 'none';
    });

    // Room: shared toggle
    $('#room-shared').addEventListener('change', function () {
      $('#room-hourly-cost-field').style.display = this.checked ? '' : 'none';
    });

    // Room filters
    var filterDebounce = null;
    function onRoomFilter() {
      clearTimeout(filterDebounce);
      filterDebounce = setTimeout(loadRooms, 400);
    }
    $('#rooms-search').addEventListener('input', onRoomFilter);
    $('#rooms-node-filter').addEventListener('change', loadRooms);
    $('#rooms-status-filter').addEventListener('change', loadRooms);

    // Rooms: assign owner delegation
    $('#rooms-tbody').addEventListener('click', function (e) {
      var btn = e.target.closest('.assign-btn');
      if (btn) openAssignModal(btn.dataset.id, btn.dataset.name);
    });

    // Assign modal
    $('#assign-modal-close').addEventListener('click', closeAssignModal);
    $('#assign-modal').addEventListener('click', function (e) {
      if (e.target === this) closeAssignModal();
    });

    var assignDebounce = null;
    $('#assign-search').addEventListener('input', function () {
      var val = this.value;
      clearTimeout(assignDebounce);
      assignDebounce = setTimeout(function () { renderAssignUsers(val); }, 300);
    });

    // Assign modal: assign delegation
    $('#assign-users-tbody').addEventListener('click', function (e) {
      var btn = e.target.closest('.do-assign-btn');
      if (btn) doAssign(btn.dataset.uid);
    });

    // Shard modal
    $('#shard-modal-close').addEventListener('click', closeShardModal);
    $('#shard-modal-cancel').addEventListener('click', closeShardModal);
    $('#shard-modal').addEventListener('click', function (e) {
      if (e.target === this) closeShardModal();
    });
    $('#shard-modal-submit').addEventListener('click', submitShardModal);

    // Users: add/remove shards delegation
    $('#users-tbody').addEventListener('click', function (e) {
      var addBtn = e.target.closest('.add-shards-btn');
      if (addBtn) {
        openShardModal(addBtn.dataset.uid, addBtn.dataset.uname, 'add');
        return;
      }
      var removeBtn = e.target.closest('.remove-shards-btn');
      if (removeBtn) {
        openShardModal(removeBtn.dataset.uid, removeBtn.dataset.uname, 'remove');
      }
    });

    // Users search
    var usersDebounce = null;
    $('#users-search').addEventListener('input', function () {
      clearTimeout(usersDebounce);
      usersDebounce = setTimeout(renderUsers, 300);
    });
  }

  // ---- Init ----

  function init() {
    bindEvents();
    checkToken();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
