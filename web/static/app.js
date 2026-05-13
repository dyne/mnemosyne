// Fetch version on page load
fetch('/version')
  .then(function(r) { return r.json(); })
  .then(function(data) {
    var el = document.getElementById('footer-version');
    if (el) el.textContent = 'v' + data.version;
    console.log('%c mnemosyne %c v' + data.version + ' ',
      'background:#0a1628;color:#c9a96e;font-size:1em;padding:2px 6px;font-family:monospace;',
      'color:#b8c5d6;');
  })
  .catch(function() { /* use default */ });
// Mnemosyne — browser console signature
console.log(
  "%c MNEMOSYNE %c titaness of memory ",
  "background:#0a1628;color:#c9a96e;font-size:1.2em;padding:4px 8px;font-family:monospace;",
  "color:#b8c5d6;font-style:italic;"
);
console.log(
  "%c cryptographic memory archive — dyne.org ",
  "color:#8899aa;font-family:monospace;font-size:0.8em;"
);
console.log(
  "%c All cryptography delegated to Zenroom. Application code only orchestrates.",
  "color:#8899aa;font-size:0.7em;"
);

// ------ helpers ------

function api(url, method, body) {
  var opts = { method: method, headers: {} };
  if (body) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  return fetch(url, opts).then(function(r) {
    return r.json().then(function(data) {
      if (!r.ok) throw new Error(data.error || r.statusText);
      return data;
    });
  });
}

function setSpinner(name, on) {
  var el = document.getElementById(name + '-spinner');
  if (el) el.style.display = on ? 'inline' : 'none';
}

function showResult(id, ok, html) {
  var el = document.getElementById(id);
  el.innerHTML = '';
  var div = document.createElement('div');
  div.className = ok ? 'result-ok' : 'result-err';
  div.innerHTML = html;
  el.appendChild(div);
}

function escapeHtml(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

function prettyJson(obj) {
  try { return JSON.stringify(obj, null, 2); } catch(e) { return String(obj); }
}

// ------ Remember ------

function rememberMemory(e) {
  e.preventDefault();
  var raw = document.getElementById('remember-payload').value.trim();
  var payload;
  try { payload = JSON.parse(raw); } catch (err) {
    showResult('remember-result', false, 'Invalid JSON: ' + escapeHtml(err.message));
    return;
  }
  setSpinner('remember', true);
  api('/memories', 'POST', { payload: payload })
    .then(function(m) {
      var rows = [
        ['Memory ID', m.memory_id],
        ['Leaf Hash', m.leaf_hash],
        ['Inserted At', m.inserted_at],
        ['Payload', prettyJson(m.payload)]
      ];
      var html = rows.map(function(r) {
        return '<div class="hash-label">' + escapeHtml(r[0]) + '</div>' +
               '<div class="hash">' + escapeHtml(r[1]) + '</div>';
      }).join('');
      // Also auto-fill the recall field so the user can immediately recall
      document.getElementById('recall-id').value = m.memory_id;
      html += '<p style="margin-top:0.75rem;color:var(--brass);font-size:0.8rem;">&#8594; Memory ID copied to Recall tab.</p>';
      showResult('remember-result', true, html);
    })
    .catch(function(err) { showResult('remember-result', false, escapeHtml(err.message)); })
    .finally(function() { setSpinner('remember', false); });
}

// ------ Recall ------

function recallMemory(e) {
  e.preventDefault();
  var id = document.getElementById('recall-id').value.trim();
  if (!id) return;
  setSpinner('recall', true);
  api('/memories/' + encodeURIComponent(id), 'GET')
    .then(function(m) {
      var rows = [
        ['Memory ID', m.memory_id],
        ['Leaf Hash', m.leaf_hash],
        ['Beacon ID', m.beacon_id],
        ['Inserted At', m.inserted_at],
        ['Payload', prettyJson(m.payload)]
      ];
      var html = rows.map(function(r) {
        return '<div class="hash-label">' + escapeHtml(r[0]) + '</div>' +
               '<div class="hash">' + escapeHtml(r[1]) + '</div>';
      }).join('');
      showResult('recall-result', true, html);
    })
    .catch(function(err) { showResult('recall-result', false, escapeHtml(err.message)); })
    .finally(function() { setSpinner('recall', false); });
}

// ------ Route ------

function generateRoute(e) {
  e.preventDefault();
  var id = document.getElementById('route-memory-id').value.trim();
  if (!id) return;
  setSpinner('route', true);
  api('/proofs/' + encodeURIComponent(id), 'GET')
    .then(function(proof) {
      var html = '<div class="hash-label">Leaf</div><div class="hash">' + escapeHtml(proof.leaf) + '</div>';
      html += '<div class="hash-label">Root (Constellation)</div><div class="hash">' + escapeHtml(proof.root) + '</div>';
      html += '<div class="hash-label">Position / Leaf Count</div><div class="hash">' + proof.position + ' / ' + proof.leaf_count + '</div>';
      html += '<div class="hash-label">Proof Path (' + proof.path.length + ' elements)</div>';
      for (var i = 0; i < proof.path.length; i++) {
        html += '<div class="hash">[' + i + '] ' + escapeHtml(proof.path[i]) + '</div>';
      }
      // Auto-fill witness form
      var witnessData = { leaf: proof.leaf, root: proof.root, path: proof.path, position: proof.position, leaf_count: proof.leaf_count };
      document.getElementById('witness-data').value = prettyJson(witnessData);
      html += '<p style="margin-top:0.75rem;color:var(--brass);font-size:0.8rem;">&#8594; Witness form auto-filled — switch to Witness tab to verify.</p>';
      showResult('route-result', true, html);
    })
    .catch(function(err) { showResult('route-result', false, escapeHtml(err.message)); })
    .finally(function() { setSpinner('route', false); });
}

// ------ Witness ------

function witnessProof(e) {
  e.preventDefault();
  var raw = document.getElementById('witness-data').value.trim();
  var data;
  try { data = JSON.parse(raw); } catch (err) {
    showResult('witness-result', false, 'Invalid JSON: ' + escapeHtml(err.message));
    return;
  }
  setSpinner('witness', true);
  api('/verify', 'POST', data)
    .then(function(result) {
      var cls = result.valid ? 'valid' : 'invalid';
      var label = result.valid ? 'VALID — The proof is authentic' : 'INVALID — The proof does not verify';
      var html = '<div class="hash ' + cls + '">' + escapeHtml(label) + '</div>';
      html += '<div class="hash-label">Leaf</div><div class="hash">' + escapeHtml(result.leaf) + '</div>';
      html += '<div class="hash-label">Root</div><div class="hash">' + escapeHtml(result.root) + '</div>';
      showResult('witness-result', result.valid, html);
    })
    .catch(function(err) { showResult('witness-result', false, escapeHtml(err.message)); })
    .finally(function() { setSpinner('witness', false); });
}

// ------ Beacon ------

function anchorBeacon(e) {
  e.preventDefault();
  setSpinner('beacon', true);
  api('/checkpoints', 'POST')
    .then(function(beacon) {
      var html = '<div class="hash-label">Beacon ID</div><div class="hash">' + escapeHtml(beacon.beacon_id) + '</div>';
      html += '<div class="hash-label">Root</div><div class="hash">' + escapeHtml(beacon.root) + '</div>';
      showResult('beacon-result', true, html);
    })
    .catch(function(err) { showResult('beacon-result', false, escapeHtml(err.message)); })
    .finally(function() { setSpinner('beacon', false); });
}

// ------ Contracts ------

function loadContracts() {
  var listEl = document.getElementById('contracts-list');
  var sourceEl = document.getElementById('contracts-source');
  listEl.innerHTML = '<p style="color:var(--text-dim);">Loading contracts...</p>';
  sourceEl.innerHTML = '';
  api('/contracts', 'GET')
    .then(function(data) {
      var html = '<p style="color:var(--text-dim);margin-bottom:0.5rem;">' + data.contracts.length + ' contracts in <code>' + escapeHtml(data.directory) + '</code></p>';
      data.contracts.forEach(function(c) {
        var lang = c.language === 'lua' ? 'Lua' : 'Zencode';
        html += '<div class="contract-item" style="cursor:pointer;padding:0.4rem 0.75rem;margin:0.25rem 0;background:var(--deep-navy);border-left:2px solid var(--brass);" onclick="viewContract(\'' + escapeHtml(c.name) + '\')">';
        html += '<span class="hash" style="border:none;padding:0;margin:0;">' + escapeHtml(c.name) + '</span>';
        html += ' <span style="color:var(--text-dim);font-size:0.7rem;">(' + lang + ', ' + c.size + ' bytes)</span>';
        html += '</div>';
      });
      listEl.innerHTML = html;
    })
    .catch(function(err) {
      listEl.innerHTML = '<p style="color:#c4746e;">Failed to load contracts: ' + escapeHtml(err.message) + '</p>';
    });
}

function viewContract(name) {
  var sourceEl = document.getElementById('contracts-source');
  sourceEl.innerHTML = '<p style="color:var(--text-dim);">Loading ' + escapeHtml(name) + '...</p>';
  fetch('/contracts/' + encodeURIComponent(name))
    .then(function(r) {
      if (!r.ok) throw new Error('Not found');
      return r.text();
    })
    .then(function(source) {
      var html = '<div class="hash-label">' + escapeHtml(name) + '</div>';
      html += '<pre class="hash" style="white-space:pre-wrap;max-height:500px;overflow-y:auto;">' + escapeHtml(source) + '</pre>';
      sourceEl.innerHTML = html;
    })
    .catch(function(err) {
      sourceEl.innerHTML = '<p style="color:#c4746e;">Failed to load: ' + escapeHtml(err.message) + '</p>';
    });
}

// ------ Beacon (updated) ------

function anchorBeacon(e) {
  e.preventDefault();
  setSpinner('beacon', true);
  document.getElementById('beacon-result').innerHTML = '';
  document.getElementById('beacon-memories').innerHTML = '';
  api('/checkpoints', 'POST')
    .then(function(beacon) {
      var html = '<div class="hash ok">Beacon anchored</div>';
      html += '<div class="hash-label">Beacon ID</div><div class="hash">' + escapeHtml(beacon.beacon_id) + '</div>';
      html += '<div class="hash-label">Root</div><div class="hash">' + escapeHtml(beacon.root) + '</div>';
      html += '<div class="hash-label">Memories sealed</div><div class="hash">' + beacon.proof_count + '</div>';
      if (beacon.parent_beacon_id) {
        html += '<div class="hash-label">Parent Beacon</div><div class="hash"><a href="#" onclick="lookupBeaconById(\'' + escapeHtml(beacon.parent_beacon_id) + '\');return false;" style="color:var(--brass);">' + escapeHtml(beacon.parent_beacon_id) + '</a></div>';
      }
      document.getElementById('beacon-result').innerHTML = html;
      // Also show the memories in this beacon
      lookupBeaconById(beacon.beacon_id);
    })
    .catch(function(err) { document.getElementById('beacon-result').innerHTML = '<div class="result-err">' + escapeHtml(err.message) + '</div>'; })
    .finally(function() { setSpinner('beacon', false); });
}

function lookupBeacon(e) {
  e.preventDefault();
  var id = document.getElementById('beacon-lookup-id').value.trim();
  if (!id) return;
  lookupBeaconById(id);
}

function lookupBeaconById(id) {
  setSpinner('beacon', true);
  document.getElementById('beacon-result').innerHTML = '';
  document.getElementById('beacon-memories').innerHTML = '';
  // Fetch beacon details
  api('/beacons/' + encodeURIComponent(id), 'GET')
    .then(function(beacon) {
      var html = '<div class="hash-label">Beacon ID</div><div class="hash">' + escapeHtml(beacon.beacon_id) + '</div>';
      html += '<div class="hash-label">Root</div><div class="hash">' + escapeHtml(beacon.root) + '</div>';
      html += '<div class="hash-label">Memories sealed</div><div class="hash">' + beacon.proof_count + '</div>';
      if (beacon.parent_beacon_id) {
        html += '<div class="hash-label">Parent Beacon</div><div class="hash"><a href="#" onclick="lookupBeaconById(\'' + escapeHtml(beacon.parent_beacon_id) + '\');return false;" style="color:var(--brass);">' + escapeHtml(beacon.parent_beacon_id) + '</a></div>';
      } else {
        html += '<div class="hash-label">Parent Beacon</div><div class="hash" style="color:var(--text-dim);">none (genesis beacon)</div>';
      }
      html += '<div class="hash-label">Created</div><div class="hash">' + escapeHtml(beacon.created_at) + '</div>';
      document.getElementById('beacon-result').innerHTML = html;
      // Fetch memories in this beacon
      return api('/beacons/' + encodeURIComponent(id) + '/memories', 'GET');
    })
    .then(function(data) {
      if (!data) return;
      var html = '<div class="hash-label">Leaves in this constellation (' + data.memories.length + ')</div>';
      if (data.memories.length === 0) {
        html += '<div class="hash" style="color:var(--text-dim);">No memories in this beacon.</div>';
      } else {
        data.memories.forEach(function(m) {
          html += '<div class="hash" style="cursor:pointer;margin:0.25rem 0;" onclick="document.getElementById(\'route-memory-id\').value=\'' + escapeHtml(m.memory_id) + '\';showSection(\'route\');return false;" title="Click to generate proof">&#8226; ' + escapeHtml(m.memory_id) + ' (' + escapeHtml(m.leaf_hash).substring(0, 16) + '...)</div>';
        });
        html += '<p style="color:var(--text-dim);font-size:0.7rem;margin-top:0.5rem;">Click any memory to generate its proof in the Route tab.</p>';
      }
      document.getElementById('beacon-memories').innerHTML = html;
    })
    .catch(function(err) {
      if (document.getElementById('beacon-result').innerHTML === '') {
        document.getElementById('beacon-result').innerHTML = '<div class="result-err">' + escapeHtml(err.message) + '</div>';
      }
    })
    .finally(function() { setSpinner('beacon', false); });
}

// ------ Zencode syntax highlighting for highlight.js ------

hljs.registerLanguage('zencode', function(hljs) {
  return {
    name: 'Zencode',
    case_insensitive: false,
    keywords: {
      keyword: 'Scenario Given When Then IfWhen Foreach And rule unknown ignore',
      literal: 'true false nothing'
    },
    contains: [
      hljs.HASH_COMMENT_MODE,
      hljs.QUOTE_STRING_MODE,
      {
        className: 'string',
        begin: /'[^']*'/,
        relevance: 5
      },
      {
        className: 'number',
        begin: /\b\d+\b/
      }
    ]
  };
});

// ------ Updated viewContract with highlighting ------

var _originalViewContract = viewContract;
viewContract = function(name) {
  var sourceEl = document.getElementById('contracts-source');
  sourceEl.innerHTML = '<p style="color:var(--text-dim);">Loading ' + escapeHtml(name) + '...</p>';
  fetch('/contracts/' + encodeURIComponent(name))
    .then(function(r) {
      if (!r.ok) throw new Error('Not found');
      return r.text();
    })
    .then(function(source) {
      var ext = name.split('.').pop();
      var lang = ext === 'lua' ? 'lua' : 'zencode';
      var highlighted;
      try {
        highlighted = hljs.highlight(source, { language: lang }).value;
      } catch(e) {
        highlighted = escapeHtml(source);
      }
      var html = '<div class="hash-label">' + escapeHtml(name) + ' <span style="color:var(--text-dim);">(' + (lang === 'lua' ? 'Lua' : 'Zencode') + ')</span></div>';
      html += '<pre class="contract-source"><code class="hljs language-' + lang + '">' + highlighted + '</code></pre>';
      sourceEl.innerHTML = html;
    })
    .catch(function(err) {
      sourceEl.innerHTML = '<p style="color:#c4746e;">Failed to load: ' + escapeHtml(err.message) + '</p>';
    });
};
