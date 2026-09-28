import http from 'node:http';
import {readFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
import {URL} from 'node:url';
const ROOT=path.dirname(fileURLToPath(import.meta.url)),PORT=Number(process.env.PORT||3000);
const HL='https://api.hyperliquid.xyz/info',BN='https://fapi.binance.com';
const cache=new Map();
const N=()=>Date.now(), C=(x,a,b)=>Math.max(a,Math.min(b,x));
async function J(url,o={},ms=4500){const c=new AbortController(),t=setTimeout(()=>c.abort(),ms);try{const r=await fetch(url,{...o,signal:c.signal});if(!r.ok)throw Error(r.status+' '+r.statusText);return await r.json()}finally{clearTimeout(t)}}
async function H(b){return J(HL,{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify(b)})}
async function K(key,fn,ms){const x=cache.get(key);if(x&&N()-x.t<ms)return x.v;const v=await fn();cache.set(key,{t:N(),v});return v}
function ema(a,n){if(!a.length)return NaN;let e=a[0],k=2/(n+1);for(let i=1;i<a.length;i++)e=a[i]*k+e*(1-k);return e}
function ret(a,n){return a.length<=n?NaN:a.at(-1)/a.at(-1-n)-1}
function atr(a,n=14){if(a.length<n+1)return NaN;let s=0;for(let i=a.length-n;i<a.length;i++){const p=a[i-1],x=a[i];s+=Math.max(x.h-x.l,Math.abs(x.h-p.c),Math.abs(x.l-p.c))}return s/n}
function rv(a,n=60){const z=a.slice(-(n+1)).map(x=>x.c),r=[];for(let i=1;i<z.length;i++)r.push(Math.log(z[i]/z[i-1]));if(r.length<2)return NaN;const m=r.reduce((a,b)=>a+b,0)/r.length;return Math.sqrt(r.reduce((a,b)=>a+(b-m)**2,0)/(r.length-1))*Math.sqrt(525600)}
function book(b){if(!b?.levels||b.levels.length<2)return{};const bi=b.levels[0].slice(0,10).map(x=>({p:+x.px,s:+x.sz})),as=b.levels[1].slice(0,10).map(x=>({p:+x.px,s:+x.sz}));if(!bi.length||!as.length)return{};const bs=bi.reduce((a,x)=>a+x.s,0),ss=as.reduce((a,x)=>a+x.s,0),bid=bi[0].p,ask=as[0].p,mid=(bid+ask)/2,micro=(ask*bs+bid*ss)/(bs+ss);return{bid,ask,bs,ss,mid,spread:(ask-bid)/mid,imb:(bs-ss)/(bs+ss),micro,bias:(micro-mid)/mid}}
async function state(coin){
 const t=N(),all=await Promise.allSettled([
  K('mc',()=>H({type:'metaAndAssetCtxs'}),5000),K('mid',()=>H({type:'allMids'}),2500),
  K('book-'+coin,()=>H({type:'l2Book',coin}),1200),K('c-'+coin,()=>H({type:'candleSnapshot',req:{coin,interval:'1m',startTime:t-21600000,endTime:t}}),7000),
  K('bt-'+coin,()=>J(BN+'/fapi/v1/ticker/24hr?symbol='+coin+'USDT'),4000),
  K('oi-'+coin,()=>J(BN+'/fapi/v1/openInterest?symbol='+coin+'USDT'),5000),
  K('pr-'+coin,()=>J(BN+'/fapi/v1/premiumIndex?symbol='+coin+'USDT'),5000),
  K('kl-'+coin,()=>J(BN+'/fapi/v1/klines?symbol='+coin+'USDT&interval=1m&limit=240'),7000)
 ]);
 const V=all.map(x=>x.status==='fulfilled'?x.value:null),mc=V[0],mids=V[1],bk=V[2],cc=V[3],bt=V[4],boi=V[5],bp=V[6],bkl=V[7];
 let ctx=null;if(Array.isArray(mc)&&Array.isArray(mc[0])&&Array.isArray(mc[1])){const i=mc[0]?.universe?.findIndex(x=>x.name===coin);if(i>=0)ctx=mc[1][i]}
 const hc=(cc||[]).map(x=>({t:+x.t,o:+x.o,h:+x.h,l:+x.l,c:+x.c,v:+x.v})),bc=(bkl||[]).map(x=>({c:+x[4],q:+x[7],tb:+x[10]})),cl=hc.map(x=>x.c),d=book(bk);
 const hp=mids?.[coin]?+mids[coin]:ctx?.markPx?+ctx.markPx:NaN,bp0=bt?+bt.lastPrice:NaN;
 const funding=ctx?.funding!=null?+ctx.funding:(bp?+bp.lastFundingRate:NaN),oi=ctx?.openInterest!=null?+ctx.openInterest:(boi?+boi.openInterest:NaN);
 const flow=bc.length?bc.slice(-15).reduce((s,x)=>s+(x.q?2*x.tb/x.q-1:0),0)/Math.min(15,bc.length):NaN,eb=ema(cl.slice(-180),9)/ema(cl.slice(-180),21)-1;
 const r5=ret(cl,5),r15=ret(cl,15),r60=ret(cl,60),cross=Number.isFinite(hp)&&Number.isFinite(bp0)?hp/bp0-1:NaN;
 const blocks={
  structure:C((r5||0)*800+(r15||0)*350+(r60||0)*120+(Number.isFinite(eb)?eb*120:0),-1,1),
  flow:C((flow||0)*2.5,-1,1),
  l2:C((d.imb||0)*1.4+(d.bias||0)*3000,-1,1),
  derivatives:C(-(funding||0)*4000+(r15<0&&Number.isFinite(oi)?0.2:0),-1,1),
  crossVenue:C((cross||0)*500+C((r5||0)*500,-.8,.8),-1,1)
 };
 const w={structure:.29,flow:.22,l2:.20,derivatives:.15,crossVenue:.14},score=Object.keys(blocks).reduce((s,k)=>s+blocks[k]*w[k],0);
 const dis=Math.sqrt(Object.values(blocks).reduce((s,x)=>s+(x-score)**2,0)/5), finite=[hp,bp0,d.imb,r5,r15,r60,flow,funding,oi].filter(Number.isFinite).length,quality=Math.round(C(100*(.62*finite/9+.28*(1-C(dis,0,1))+.1*(bk?1:0)),0,100));
 const predictions={};for(const k of [5,15,60,240]){const f={5:1,15:.92,60:.74,240:.52}[k],s=score*f,neu=C(.18+dis*.42+(100-quality)*.002,.18,.72),q=1-neu,z=1/(1+Math.exp(-4.2*s)),pl=neu/2+q*z,ps=neu/2+q*(1-z),no=quality<55||Math.max(pl,ps)<.58||dis>.55;predictions[k]={direction:no?'NO TRADE':pl>ps?'LONG':'SHORT',pLong:pl,pNeutral:neu,pShort:ps,noTrade:no,score:s}}
 return{coin,asOf:t,sources:{hyperliquid:{meta:!!mc,mids:!!mids,l2:!!bk,candles:!!cc},binance:{ticker:!!bt,oi:!!boi,premium:!!bp,klines:!!bkl}},price:hp,binancePrice:bp0,funding,oi,depth:d,flow,blocks,score,disagreement:dis,quality,predictions,regime:dis>.55?'TRANSITION':Math.abs(r15||0)>.003?'TREND':Math.abs(r5||0)<.0005&&Math.abs(d.imb||0)<.12?'RANGE':'MOMENTUM',candles:hc,atr:atr(hc.slice(-120)),rv:rv(hc.slice(-120)),crossBps:Number.isFinite(cross)?cross*10000:NaN,calibration:{status:'COLD START',samples:0,brier:null,logLoss:null}}
}
function out(res,code,data,type='application/json'){res.writeHead(code,{'content-type':type,'cache-control':'no-store','access-control-allow-origin':'*'});res.end(type==='application/json'?JSON.stringify(data):data)}
async function main(req,res){try{const u=new URL(req.url,'http://x');if(u.pathname==='/health')return out(res,200,{ok:true,time:N()});if(u.pathname==='/api/state')return out(res,200,await state((u.searchParams.get('coin')||'BTC').toUpperCase().replace(/[^A-Z0-9]/g,'')));const p=u.pathname==='/'?'/public/index.html':u.pathname;if(!p.startsWith('/public/'))return out(res,404,{error:'not found'});const f=path.join(ROOT,p),b=await readFile(f),e=path.extname(f),ty=e==='.html'?'text/html; charset=utf-8':e==='.js'?'text/javascript; charset=utf-8':e==='.css'?'text/css; charset=utf-8':'application/octet-stream';return out(res,200,b.toString(),ty)}catch(e){return out(res,500,{error:String(e.message||e)})}}
http.createServer(main).listen(PORT,'0.0.0.0',()=>console.log('Crypto Direction Engine '+PORT));