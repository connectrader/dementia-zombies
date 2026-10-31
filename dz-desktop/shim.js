/* Desktop runtime: stands in for claude.ai's room, saves and downloads, backed by the local game server. */
(function(){
'use strict';
var isLocal=/^(127\.0\.0\.1|localhost|\[::1\])$/.test(location.hostname);
var HOME=location.host,enc=encodeURIComponent;
function Room(host,name){
  var ws=null,me=null,closed=false,myP={},peers=new Map(),subs=[],lastMsg=0,waiters=[],retry=0;
  function list(){var a=[];if(me)a.push({peer:me,sameTab:true,isMe:true,presence:myP});peers.forEach(function(v,k){a.push({peer:k,sameTab:false,isMe:false,presence:v||{}});});return a;}
  function notify(){var s=list();subs.slice().forEach(function(f){try{f({peers:s});}catch(e){}});}
  function send(o){if(ws&&ws.readyState===1)try{ws.send(JSON.stringify(o));}catch(e){}}
  function settle(ok){waiters.splice(0).forEach(function(f){f(ok);});}
  function connect(){
    if(closed)return;
    try{ws=new WebSocket('ws://'+host+'/ws?room='+enc(name)+(me?'&id='+me:''));}catch(e){setTimeout(connect,1500);return;}
    ws.onopen=function(){retry=0;lastMsg=Date.now();};
    ws.onmessage=function(ev){lastMsg=Date.now();var m;try{m=JSON.parse(ev.data);}catch(_){return;}
      if(m.t==='hello'){me=m.you;peers.clear();(m.peers||[]).forEach(function(q){peers.set(q.peer,q.p||{});});send({t:'p',p:myP});settle(true);notify();}
      else if(m.t==='u'){peers.set(m.peer,m.p||{});notify();}
      else if(m.t==='x'){peers.delete(m.peer);notify();}};
    ws.onclose=function(){ws=null;if(peers.size){peers.clear();notify();}if(closed)return;settle(false);setTimeout(connect,Math.min(4000,400*(++retry)));};
    ws.onerror=function(){};
  }
  var kt=setInterval(function(){if(closed){clearInterval(kt);return;}send({t:'k'});if(ws&&ws.readyState===1&&Date.now()-lastMsg>15000){try{ws.close();}catch(_){}}},5000);
  this.name=name;
  this.ready=function(){return new Promise(function(res){if(me&&ws&&ws.readyState===1){res(true);return;}waiters.push(res);setTimeout(function(){res(false);},5000);});};
  this.peers=function(){return list();};
  this.presence=function(patch){var cur=Object.assign({},myP);for(var k in patch){if(patch[k]===null)delete cur[k];else cur[k]=patch[k];}myP=cur;send({t:'p',p:cur});return Promise.resolve();};
  this.onPeers=function(f,onClose){subs.push(f);setTimeout(function(){f({peers:list()});},0);return function(){var i=subs.indexOf(f);if(i>=0)subs.splice(i,1);};};
  this.leave=function(){closed=true;clearInterval(kt);if(ws)try{ws.close();}catch(_){}peers.clear();return Promise.resolve();};
  this.emit=function(){return Promise.resolve();};this.on=function(){return function(){};};
  this.connected=function(){return!!(ws&&ws.readyState===1);};this.onConnection=function(){return function(){};};
  connect();
}
var lobby=new Room(HOME,'__lobby');
lobby.join=function(n){
  var code=String(n).replace(/^dz-/,''),host=HOME;
  lobby.peers().forEach(function(q){var d=q.presence&&q.presence.dz;if(d&&d.code===code&&d.addr&&host===HOME&&/^lan:/.test(q.peer))host=d.addr;});
  var r=new Room(host,n);
  return r.ready().then(function(ok){if(ok)return r;r.leave();var e=new Error('unreachable');e.code='unreachable';throw e;});
};
var H={'X-DZ':'1'};
function ioErr(){var e=new Error('io');e.code='io';return e;}
var db={collection:function(){return{doc:function(id){var u='/api/save/'+enc(id);return{
  get:function(){return fetch(u,{headers:H}).then(function(r){if(r.status===404)return{exists:false,data:function(){return null;}};if(!r.ok)throw ioErr();return r.json().then(function(d){return{exists:true,data:function(){return d;}};});});},
  set:function(v){return fetch(u,{method:'PUT',headers:H,body:JSON.stringify(v)}).then(function(r){if(!r.ok)throw ioErr();});},
  delete:function(){return fetch(u,{method:'DELETE',headers:H}).then(function(){});}};}};}};
var user={id:function(){return Promise.resolve('local');}};
var downloads={save:function(o){
  var data=typeof o.data==='string'?o.data:JSON.stringify(o.data);
  if(isLocal)return fetch('/api/export?name='+enc(o.filename),{method:'POST',headers:H,body:data}).then(function(r){if(!r.ok)throw ioErr();return r.json();}).then(function(j){window.__dzLastExport=j.path;});
  var a=document.createElement('a');a.href=URL.createObjectURL(new Blob([data],{type:'application/json'}));a.download=o.filename;document.body.appendChild(a);a.click();
  setTimeout(function(){URL.revokeObjectURL(a.href);a.remove();},1000);window.__dzLastExport=o.filename;return Promise.resolve();}};
window.__dzLan='';
fetch('/api/info').then(function(r){return r.json();}).then(function(j){var ip=(j.ips||[])[0];window.__dzLan=ip?ip+(j.port!==47800?':'+j.port:''):'';}).catch(function(){});
var update=isLocal?{
  check:function(force){return fetch('/api/update/check'+(force?'?force=1':''),{headers:H,cache:'no-store'}).then(function(r){return r.json();}).catch(function(){return{ok:false,err:'the game\'s service didn\'t answer'};});},
  install:function(){return fetch('/api/update/install',{method:'POST',headers:H}).then(function(r){return r.json();}).catch(function(){return{ok:false,err:'the game\'s service didn\'t answer'};});},
  restart:function(){return fetch('/api/update/restart',{method:'POST',headers:H}).then(function(r){return r.json();}).catch(function(){return{ok:false};});},
  version:function(){return fetch('/api/info',{cache:'no-store'}).then(function(r){return r.json();}).then(function(j){return j.version;}).catch(function(){return null;});}}:null;
window.claude={desktop:true,update:update,
  use:function(n){return Promise.resolve(n==='room'?lobby:n==='db'?(isLocal?db:null):n==='user'?(isLocal?user:null):n==='downloads'?downloads:null);},
  lan:{ping:function(addr){var t=performance.now();return fetch('http://'+addr+'/api/info',{cache:'no-store'}).then(function(r){return r.ok?Math.max(1,Math.round(performance.now()-t)):null;}).catch(function(){return null;});},addHost:function(addr){return fetch('/api/addhost?addr='+enc(String(addr||'').trim()),{method:'POST',headers:H}).then(function(r){return r.json();}).catch(function(){return{ok:false};});}}};
})();
