// Supply an existing Playwright installation; this evidence script adds no project dependency.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const { spawn, execFileSync } = require('node:child_process');
const fs = require('node:fs');
(async () => {
 const url = process.env.PREVIEW_URL || 'http://127.0.0.1:7119';
 const server = process.env.PREVIEW_URL ? null : spawn('task', ['design:preview'], {cwd: process.cwd(), detached: true, stdio: ['ignore', 'pipe', 'pipe']});
 server?.stderr.on('data', x=>process.stderr.write(x));
 let browser;
 let started = !server;
 server?.stderr.on('data', x => { if (x.toString().includes('Design preview:')) started = true; });
 try {
  await new Promise((resolve,reject)=>{let count=0; const timer=setInterval(async()=>{try {if (!started || (server && server.exitCode !== null)) throw Error('server not started'); await fetch(url);clearInterval(timer);resolve();}catch {if(++count>100){clearInterval(timer);reject(new Error('preview not ready'));}}},100);});
  browser = await chromium.launch({executablePath:process.env.CHROMIUM_PATH,headless:true});
  fs.mkdirSync('docs/design-preview',{recursive:true});
  const report=[];
  function check(ok,message){if(!ok)throw Error(message); report.push(message);}
  for(const viewport of [{width:320,height:568},{width:390,height:844}]){
   const page=await browser.newPage({viewport});
   const errors=[];page.on('pageerror', e=>errors.push(e.message));
   await page.goto(url); await page.evaluate(()=>document.fonts.ready);
   check(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`${viewport.width}: no page overflow`);
   check(await page.evaluate(()=>[...document.querySelectorAll('button')].filter(b=>b.getBoundingClientRect().width>0).every(b=>{const r=b.getBoundingClientRect();return r.width>=48&&r.height>=48})),`${viewport.width}: targets >=48px`);
   await page.screenshot({path:`docs/design-preview/${viewport.width}-page.png`,fullPage:true});
   await page.getByRole('button',{name:'Pay Sam'}).click();
   const field=page.getByLabel('Amount in whole dollars');
   const backgroundTop = await page.locator('main').evaluate(e=>e.getBoundingClientRect().top);
   await page.mouse.move(viewport.width / 2, 5);
   await page.mouse.wheel(0, 200);
   await page.waitForTimeout(150);
   const afterWheel = await page.locator('main').evaluate(e=>e.getBoundingClientRect().top);
   check(Math.abs(afterWheel - backgroundTop) < 1, `${viewport.width}: background stays fixed during backdrop scrolling (${backgroundTop} -> ${afterWheel})`);
   check(await field.evaluate(e=>e===document.activeElement),`${viewport.width}: initial amount focus`);
   await page.getByRole('button',{name:'$200',exact:true}).click();
   check(await field.inputValue()==='200',`${viewport.width}: preset replaces amount`);
   await page.screenshot({path:`docs/design-preview/${viewport.width}-sheet.png`});
   await field.fill('123'); await field.evaluate(e=>e.setSelectionRange(1,2));
   await page.getByRole('button',{name:'9',exact:true}).click();
   check(await field.inputValue()==='193',`${viewport.width}: pointer selection insertion`);
   await page.getByRole('button',{name:'Backspace',exact:true}).click();
   check(await field.inputValue()==='13',`${viewport.width}: backspace`);
   await field.fill('');await field.pressSequentially('50');
   check(await field.inputValue()==='50',`${viewport.width}: physical typing works`);
   await page.context().grantPermissions(['clipboard-read','clipboard-write']);
   await page.evaluate(()=>navigator.clipboard.writeText('1.50'));
   await field.press('Control+a');await field.press('Control+v');
   check(await field.inputValue()==='1.50',`${viewport.width}: invalid clipboard paste remains unchanged`);
   check(await page.getByRole('button',{name:'Pay amount',exact:true}).isDisabled(),`${viewport.width}: invalid edit disables submission`);
   await field.fill('18446744073709551615');
   check(await page.getByRole('button',{name:'Pay $18,446,744,073,709,551,615',exact:true}).isEnabled(),`${viewport.width}: exact uint64 maximum valid`);
   for(let i=0;i<20;i++){await page.keyboard.press('Tab');check(await page.evaluate(()=>document.activeElement===document.body || document.querySelector('dialog').contains(document.activeElement)),`${viewport.width}: Tab does not focus background controls ${i}`);}
   await page.keyboard.press('Escape');
   check(await page.getByRole('button',{name:'Pay Sam'}).evaluate(e=>e===document.activeElement),`${viewport.width}: Escape restores focus`);
   check(Math.abs(await page.locator('main').evaluate(e=>e.getBoundingClientRect().top) - backgroundTop) < 1,`${viewport.width}: close restores background position`);
   await page.getByRole('button',{name:'Pay Sam'}).click();check(await field.inputValue()==='',`${viewport.width}: reopen clears amount`);
   await field.fill('200');await page.getByRole('button',{name:'Pay $200',exact:true}).click();
   check((await page.getByRole('status').textContent()).includes('No money moved'),`${viewport.width}: demo submit stays local`);
   await page.keyboard.press('Escape');
   await page.evaluate(()=>document.documentElement.style.fontSize='200%');
   check(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`${viewport.width}: 200% text no page overflow`);
   await page.screenshot({path:`docs/design-preview/${viewport.width}-large-text.png`,fullPage:true});
   await page.getByRole('button',{name:'Pay Sam'}).click();await field.fill('18446744073709551615');
   check(await page.locator('dialog').evaluate(e=>e.scrollWidth<=e.clientWidth),`${viewport.width}: enlarged sheet no horizontal overflow`);
   await page.screenshot({path:`docs/design-preview/${viewport.width}-large-sheet.png`});
   await page.setViewportSize({width:viewport.width,height:350});
   await page.getByRole('button',{name:'Pay $18,446,744,073,709,551,615',exact:true}).scrollIntoViewIfNeeded();
   check(await page.getByRole('button',{name:'Pay $18,446,744,073,709,551,615',exact:true}).isVisible(),`${viewport.width}: short sheet submit scrolls into view`);
   const sheetScroll = await page.locator('dialog').evaluate(e=>e.scrollTop);
   await page.mouse.move(viewport.width / 2, 330);
   await page.mouse.wheel(0, -100);
   await page.waitForTimeout(150);
   check(await page.locator('dialog').evaluate(e=>e.scrollTop) < sheetScroll,`${viewport.width}: sheet remains scrollable while background is locked`);
   const lockedTop = await page.locator('main').evaluate(e=>e.getBoundingClientRect().top);
   await page.locator('dialog').evaluate(e=>{e.scrollTop=e.scrollHeight;});
   await page.mouse.wheel(0, 200);
   await page.waitForTimeout(150);
   check(Math.abs(await page.locator('main').evaluate(e=>e.getBoundingClientRect().top) - lockedTop) < 1,`${viewport.width}: sheet boundary scrolling does not move background`);
   await page.emulateMedia({reducedMotion:'reduce'});
   check(await page.evaluate(()=>matchMedia('(prefers-reduced-motion: reduce)').matches),`${viewport.width}: reduced motion preference active`);
   check(errors.length===0,`${viewport.width}: no browser JS errors`);
   await page.close();
  }
  fs.writeFileSync('docs/design-preview/browser-checks.txt',report.join('\n')+'\n');
  console.log('PASS',report.length,'browser assertions');
 } finally {
  if(browser)await browser.close();
  if (server) {
  const rows = execFileSync('ps', ['-eo','pid=,ppid='], {encoding:'utf8'}).trim().split('\n').map(line=>line.trim().split(/\s+/).map(Number));
  const owned = new Set([server.pid]);
  for (let pass=0;pass<rows.length;pass++) for (const [pid,parent] of rows) if(owned.has(parent)) owned.add(pid);
  for(const pid of [...owned].reverse()) { try { process.kill(pid,'SIGTERM'); } catch(e) { if(e.code!=='ESRCH') throw e; } }
  }
 }
})().catch(e=>{console.error(e);process.exitCode=1});
