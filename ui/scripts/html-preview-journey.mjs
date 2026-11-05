// Real-app Chromium boundary checks for FS-03.R109; no external requests escape.
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { createServer } from 'vite';
import { chromium } from 'playwright';
const out = process.argv[2] || '/tmp/chuck-html-preview-journey';
mkdirSync(out, { recursive: true });
const checks = [], requests = [], launches = [];
const check = (name, pass, detail) => { checks.push({ name, pass: !!pass, detail }); console.log(pass ? 'PASS' : 'FAIL', name, detail ?? ''); };
const content = `<!doctype html><html><head><style>body{background:rgb(12,34,56);color:white;font:19px monospace}button{background:rgb(220,180,90)}</style><link rel="stylesheet" href="relative.css"><script src="https://preview.invalid/remote.js"></script></head><body><h1>Independent HTML</h1><button id="counter" onclick="this.textContent=++window.count">0</button><img src="relative.png"><script>window.count=0;try{parent.document.body;window.parentRead=true}catch{window.parentRead=false}try{localStorage.getItem('x');window.storageRead=true}catch{window.storageRead=false}fetch('https://preview.invalid/fetch').catch(()=>{});parent.postMessage({type:'open-in-browser',path:'evil.html'},'*')</script></body></html>`;
const agent = {agent_id:'a1',name:'HTML demo',role:'implementer',project:'demo',backend:'claude',model:'m',interface:'chat',created_at:'2026-11-05T10:00:00Z',running:true,state:'idle',archived:false};
const room = JSON.parse(readFileSync(new URL('../src/features/thinktank/fixtures/room.json', import.meta.url), 'utf8'));
const server = await createServer({server:{host:'127.0.0.1',port:5198,strictPort:true},logLevel:'error'});
let browser;
try {
 await server.listen();
 browser = await chromium.launch().catch(()=>chromium.launch({executablePath:'/opt/pw-browsers/chromium'}));
 const context = await browser.newContext({viewport:{width:1440,height:900}});
 await context.addInitScript((agent)=>{delete window.SharedWorker;window.EventSource=class extends EventTarget{constructor(){super();setTimeout(()=>{this.onopen?.();for(const data of [{agent_id:agent.agent_id,data:agent},{agent_id:'a2',data:{...agent,agent_id:'a2',running:false,archived:true}},{agent_id:'__hydrated__',data:{hydrated:true}}])this.dispatchEvent(new MessageEvent('state_update',{data:JSON.stringify(data)}));});}close(){}};},agent);
 await context.route('**/*',async route=>{
  const req=route.request(),u=new URL(req.url()),p=u.pathname;
  if(u.hostname!=='127.0.0.1'&&u.hostname!=='localhost'){requests.push(req.url());return route.abort();}
  if(!p.startsWith('/api/')){if(p.endsWith('relative.css')||p.endsWith('relative.png')){requests.push(req.url());return route.abort();}return route.continue();}
  let json={};
  if(p==='/api/config')json={onboarding_complete:true};
  if(p==='/api/backends')json={backends:{}};
  if(p==='/api/tasks')json={tasks:[]};
  if(p.endsWith('/available-commands'))json={commands:[]};
  if(p.endsWith('/prompt'))return route.fulfill({status:404,json:{error:{code:'not_found',message:'none'}}});
  if(p==='/api/projects')json={demo:{title:'Demo',archived:false,color:[90,130,210]}};
  if(p==='/api/layout')json={order:[],density:{perRow:2,gap:16},expanded:[]};
  if(p==='/api/think-tanks')json={rooms:[],clipped:false};
  if(p.endsWith('/transcript'))json={agent_id:'a1',has_more:false,events:[]};
  if(p.endsWith('/files'))json={agent_id:'a1',files:[]};
  if(p.endsWith('/file'))json={agent_id:'a1',path:'preview.html',content,size:content.length,line_count:1,language:'html',truncated:false};
  if(p==='/api/roles')json={impl:{title:'Impl'}};
  if(p==='/api/config-sources')json=[];
  if(p==='/api/think-tanks/tt_fixture')json=room.room;
  if(p==='/api/think-tanks/tt_fixture/entries')json={version:1,entries:[{...room.entries[0],agent_id:'a_gone',body:'[HTML page](preview.html)',attempt_id:'tta_deleted'}],complete:true};
  if(p==='/api/think-tanks/tt_fixture/activity')json={version:1,activity:[],complete:true};
  if(p==='/api/think-tanks/tt_fixture/commands')json={sources:[],commands:[]};
  if(p==='/api/think-tanks/tt_fixture/files')json={version:1,sources:[{source_id:'src_23',agent_id:'a_gone',agent_name:'Gone',project:'alpha',attempt_ids:['tta_deleted']}],files:[]};
  if(/open.*browser/.test(p)){launches.push({path:p,body:req.postDataJSON()});json={path:'preview.html'};}
  return route.fulfill({json});
 });
 const page=await context.newPage();
 page.on('pageerror',error=>console.log('PAGEERROR',error.message));
 await page.goto('http://127.0.0.1:5198/agent/a1?file=preview.html');
 await page.locator('iframe.file-viewer-html').waitFor({timeout:15000}).catch(async error=>{console.log((await page.locator('body').innerText()).slice(0,6000));await page.screenshot({path:`${out}/failure.png`});throw error;});
 const getFrame=()=>page.locator('iframe.file-viewer-html').contentFrame();
 await getFrame().locator('#counter').waitFor();
 check('default Rendered',await page.getByRole('button',{name:'Rendered',exact:true}).getAttribute('aria-pressed')==='true');
 check('independent inline CSS',await getFrame().locator('body').evaluate(el=>getComputedStyle(el).backgroundColor)==='rgb(12, 34, 56)');
 await getFrame().locator('#counter').click();
 check('inline JavaScript',await getFrame().locator('#counter').textContent()==='1');
 check('opaque sandbox blocks parent and storage',await getFrame().locator('body').evaluate(()=>!window.parentRead&&!window.storageRead));
 check('CSP blocks external and relative requests',requests.length===0,requests);
 check('postMessage does not launch browser',launches.length===0);
 await page.getByRole('button',{name:'Source',exact:true}).click();
 check('Source exact text',await page.locator('.file-viewer-body').innerText().then(text=>text.includes(content)));
 await page.getByRole('button',{name:'Rendered',exact:true}).click();
 await getFrame().locator('#counter').click();
 const reloaded=page.waitForResponse(response=>new URL(response.url()).pathname.endsWith('/file'));
 await page.getByRole('button',{name:'Reload',exact:true}).click();
 await reloaded;
 await getFrame().locator('#counter').filter({hasText:/^0$/}).waitFor({timeout:5000});
 check('Reload resets script state',await getFrame().locator('#counter').textContent()==='0');
 for(const width of [1440,1024]){
  await page.setViewportSize({width,height:900});
  const fits=()=>page.locator('iframe.file-viewer-html').evaluate(el=>{const r=el.getBoundingClientRect(),p=el.parentElement.getBoundingClientRect();return r.width>200&&r.height>200&&r.right<=p.right+1&&r.bottom<=p.bottom+1&&document.documentElement.scrollWidth<=innerWidth;});
  check(`docked fits ${width}`,await fits());
  await page.screenshot({path:`${out}/docked-${width}.png`,animations:'disabled'});
  await page.getByRole('button',{name:'Focus file',exact:true}).click();
  check(`Focus fits ${width}`,await fits());
  await getFrame().locator('h1').click({button:'right'});
  const menu=page.locator('.file-focus .context-menu');
  await menu.waitFor();
  check(`frame context menu in Focus ${width}`,await menu.getByRole('button',{name:'Copy path',exact:true}).count()===1&&await menu.getByRole('button',{name:'Open in browser',exact:true}).count()===1);
  await page.screenshot({path:`${out}/focus-menu-${width}.png`,animations:'disabled'});
  await menu.getByRole('button',{name:'Open in browser',exact:true}).click();
  check(`explicit browser dispatch ${width}`,launches.at(-1)?.body?.path==='preview.html',launches.at(-1));
  await page.keyboard.press('Escape');
 }
 // Shared viewer callers must preserve their own file-source routing.
 for (const surface of [
  {name:'archive',url:'/archive/a2?file=preview.html',target:'/api/sessions/a2/file/open-browser'},
  {name:'retained room source',url:'/think-tank/tt_fixture?file=preview.html&source=src_23',target:'/api/think-tanks/tt_fixture/sources/src_23/file/open-browser'},
 ]) {
  await page.goto(`http://127.0.0.1:5198${surface.url}`);
  await getFrame().locator('#counter').waitFor();
  await getFrame().locator('#counter').click();
  check(`${surface.name}: rendered inline interaction`,await getFrame().locator('#counter').textContent()==='1');
  await getFrame().locator('h1').click({button:'right'});
  const menu=page.locator('.context-menu');
  await menu.getByRole('button',{name:'Open in browser',exact:true}).click();
  check(`${surface.name}: correct original-path dispatch`,launches.at(-1)?.path===surface.target&&launches.at(-1)?.body?.path==='preview.html',launches.at(-1));
  await page.screenshot({path:`${out}/${surface.name.replaceAll(' ','-')}.png`,animations:'disabled'});
 }
 await context.close();
} catch(error){check('journey completed',false,String(error));}
finally {await browser?.close();await server.close();writeFileSync(`${out}/report.json`,JSON.stringify({checks,requests,launches},null,2));}
if(checks.some(c=>!c.pass))process.exitCode=1;
