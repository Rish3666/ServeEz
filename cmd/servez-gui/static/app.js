const $ = (selector) => document.querySelector(selector);
const esc = (value) => String(value ?? '').replace(/[&<>"']/g, (char) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
const asNumber = (value, fallback = 0) => Number.isFinite(Number(value)) ? Number(value) : fallback;

async function request(path, options = {}) {
  const response = await fetch(`/api/${path}`, options);
  if (!response.ok) throw new Error(await response.text() || response.statusText);
  return response.json();
}

function objectValue(value, key) {
  return value && typeof value === 'object' ? value[key] : undefined;
}

function renderState(payload) {
  const objects = payload.objects || [];
  const nodes = objects.filter((object) => object.kind === 'Node');
  const workloads = objects.filter((object) => object.kind === 'Workload');
  const healthy = nodes.filter((node) => ['healthy', 'degraded'].includes(objectValue(node.status, 'state'))).length;
  const running = workloads.filter((workload) => ['running', 'scheduled'].includes(objectValue(workload.status, 'state'))).length;
  const utilization = nodes.length ? nodes.reduce((sum, node) => sum + asNumber(objectValue(objectValue(node.status, 'resources'), 'cpu_pct')), 0) / nodes.length : 0;
  $('#metrics').innerHTML = `<div class="metric"><label>NODES ONLINE</label><strong>${healthy}<span>/ ${nodes.length}</span></strong></div><div class="metric"><label>WORKLOADS ACTIVE</label><strong>${running}<span>/ ${workloads.length}</span></strong></div><div class="metric"><label>AVG CPU LOAD</label><strong>${utilization.toFixed(1)}<span>%</span></strong></div><div class="metric"><label>OPEN SIGNALS</label><strong id="metric-alerts">${countAlerts(nodes, workloads)}</strong></div>`;
  $('#node-count').textContent = `${nodes.length} NODES`;
  $('#service-count').textContent = `${running} RUNNING`;
  $('#nodes').innerHTML = nodes.length ? nodes.map(renderNode).join('') : '<div class="empty">No nodes registered.</div>';
  $('#services').innerHTML = workloads.length ? workloads.map(renderWorkload).join('') : '<tr><td colspan="5" class="empty">No workloads declared.</td></tr>';
  renderAlerts(nodes, workloads);
}

function countAlerts(nodes, workloads) {
  return nodes.filter((node) => ['degraded', 'unhealthy', 'disconnected', 'cordoned'].includes(objectValue(node.status, 'state'))).length + workloads.filter((workload) => objectValue(workload.status, 'state') === 'unschedulable').length;
}

function renderNode(node) {
  const status = node.status || {};
  const state = status.state || 'unknown';
  const score = asNumber(status.health_score);
  return `<div class="node-card ${esc(state)}"><span class="node-score">${score}</span><div class="node-name">${esc(node.name)}</div><div class="node-state">${esc(state.toUpperCase())}</div></div>`;
}

function renderWorkload(workload) {
  const spec = workload.spec || {}, status = workload.status || {};
  const state = status.state || 'declared';
  return `<tr><td>${esc(workload.name)}</td><td>${esc(spec.image || '—')}</td><td>${asNumber(status.running_replicas)}/${asNumber(spec.replicas)}</td><td><span class="state ${esc(state)}">${esc(state)}</span></td><td>${esc(status.assigned_node || '—')}</td></tr>`;
}

function renderAlerts(nodes, workloads) {
  const alerts = [];
  nodes.forEach((node) => { const state = objectValue(node.status, 'state'); if (['unhealthy','disconnected','cordoned'].includes(state)) alerts.push(['bad', `Node ${node.name} is ${state}`, 'immediate attention']); else if (state === 'degraded') alerts.push(['warn', `Node ${node.name} is degraded`, 'health below normal']); });
  workloads.forEach((workload) => { if (objectValue(workload.status, 'state') === 'unschedulable') alerts.push(['bad', `Workload ${workload.name} is unschedulable`, 'scheduler needs attention']); });
  $('#alert-count').textContent = `${alerts.length} OPEN`;
  $('#alerts').innerHTML = alerts.length ? alerts.map(([level, title, detail]) => `<div class="alert ${level}"><i class="alert-dot"></i><div>${esc(title)}<small>${esc(detail)}</small></div></div>`).join('') : '<div class="empty">All systems within bounds.</div>';
}

async function refresh() {
  try {
    const [nodePayload, workloadPayload] = await Promise.all([request('state?type=Node'), request('state?type=Workload')]);
    renderState({objects: [...(nodePayload.objects || []), ...(workloadPayload.objects || [])]});
    $('#connection').textContent = 'CONTROL LINK LIVE';
    $('.live i').style.background = 'var(--mint)';
    $('#updated').textContent = new Date().toLocaleTimeString([], {hour12:false});
  } catch (error) {
    $('#connection').textContent = 'CONTROL LINK LOST';
    $('.live i').style.background = 'var(--red)';
    $('#updated').textContent = 'RETRYING';
  }
}

$('#cost-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(event.target);
  const result = $('#cost-result');
  result.innerHTML = '<p class="empty">Comparing the baseline catalog...</p>';
  try {
    const report = await request('cost/compare', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({vcpu:asNumber(form.get('vcpu')), mem_gb:asNumber(form.get('mem_gb')), runtime_pct:asNumber(form.get('runtime_pct'))})});
    const best = report.best || {};
    result.innerHTML = `<div class="cost-best"><div><strong>$${asNumber(best.est_monthly).toFixed(2)}<small>/ month via ${esc(best.provider)} · ${esc(best.instance_type)}</small></strong></div><span class="saving">${asNumber(report.potential_savings_pct).toFixed(1)}% saved</span></div>${(report.providers || []).map((provider) => `<div class="price-row"><span>${esc(provider.provider)} / ${esc(provider.recommended)}</span><span>$${asNumber(provider.est_monthly).toFixed(2)}</span></div>`).join('')}`;
  } catch (error) { result.innerHTML = `<p class="error">${esc(error.message)}</p>`; }
});

$('#chat-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const input = $('#chat-input'), command = input.value.trim();
  if (!command) return;
  $('#chat-log').insertAdjacentHTML('beforeend', `<div class="chat-line"><b>› ${esc(command)}</b></div>`);
  input.value = '';
  const parts = command.split(/\s+/), type = parts[0], target = parts[1];
  const action = {type, target: target ? (type === 'scale' ? `workload:${target}` : target) : '', parameters: type === 'scale' ? {replicas:asNumber(parts[2])} : {}, initiator:'human:gui', confidence:0.9};
  if (!['scale','restart','stop','remove','start'].includes(type) || !target) { $('#chat-log').insertAdjacentHTML('beforeend', '<div class="chat-line reply">Read-only parser supports scale, restart, stop, remove, and start.</div>'); return; }
  try { const result = await request('simulate', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({action, simulate_only:true})}); $('#chat-log').insertAdjacentHTML('beforeend', `<div class="chat-line reply">Simulation: ${esc(result.recommendation || 'no recommendation')}.</div>`); } catch (error) { $('#chat-log').insertAdjacentHTML('beforeend', `<div class="chat-line reply">Simulation unavailable: ${esc(error.message)}</div>`); }
});

refresh();
setInterval(refresh, 2500);
