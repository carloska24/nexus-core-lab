/** Local, deterministic artwork. No tiles, remote maps or reference-image crops. */
export function UrbanMap() {
  const streets: string[] = [];
  const buildings: {x:number;y:number;w:number;h:number}[]=[];
  let seed=4217;
  const random=()=>{seed=(seed*1664525+1013904223)>>>0;return seed/4294967296;};
  for(let district=0;district<60;district++){
    const cx=random()*850,cy=random()*480,angle=random()*2.4;
    const ux=Math.cos(angle),uy=Math.sin(angle),vx=-uy,vy=ux;
    const spacing=5+random()*6,length=38+random()*95;
    for(let j=-5;j<6;j++){
      const x=cx+vx*j*spacing,y=cy+vy*j*spacing;
      streets.push(`M${x-ux*length/2},${y-uy*length/2} Q${x+random()*5},${y+random()*5} ${x+ux*length/2},${y+uy*length/2}`);
    }
    for(let j=-4;j<5;j++){
      const x=cx+ux*j*spacing,y=cy+uy*j*spacing;
      streets.push(`M${x-vx*length/2},${y-vy*length/2} L${x+vx*length/2},${y+vy*length/2}`);
    }
  }
  for(let i=0;i<1100;i++)buildings.push({x:random()*850,y:random()*480,w:1+random()*4,h:1+random()*5});
  return <g className="urban-art"><rect width="850" height="480" fill="#06131f"/>
    <g fill="#122639" opacity=".55">{buildings.map((b,i)=><rect key={i} {...b}/>)}</g>
    <g fill="none" stroke="#20374b" strokeWidth=".65" opacity=".68">{streets.map((d,i)=><path key={i} d={d}/>)}</g>
    <g fill="none" stroke="#294257" strokeWidth="1.15" opacity=".75">
      <path d="M-20 330C130 280 210 198 346 201S610 328 870 200M80-20C188 104 278 146 386 242S629 349 850 418M-20 144C178 199 320 145 456 156S730 82 870 106M270-20C220 120 271 218 354 314S492 432 510 500M-20 397C214 421 271 280 401 260S636 221 850 263M596-20C546 114 522 204 546 315S678 395 770 500"/>
    </g>
    <g fill="#102738" opacity=".8"><path d="M65 15l24 8 6 19-17 12-21-11-8-15zM729 29l13-8 12 14-3 19-19 7-11-12zM675 394l18-21 23 9 7 28-16 18-23-7zM119 359l16-16 19 10 4 20-15 12-22-8z"/></g>
  </g>;
}

export function ReferenceIcon({name}:{name:string}) {
  const users=<><circle cx="12" cy="6" r="3.3"/><circle cx="4.3" cy="8" r="2.4"/><circle cx="19.7" cy="8" r="2.4"/><path d="M6 22v-6c0-6 12-6 12 0v6zM0 19v-5c0-3 3-4 5-3l-1 8zM20 19l-1-8c2-1 5 0 5 3v5z"/></>;
  const file=<><rect x="4" y="1" width="17" height="23" rx="2"/><path d="M8 7h9M8 12h9M8 17h9" stroke="#091626" strokeWidth="2.4"/></>;
  const phone=<><rect x="5" y="1" width="14" height="22" rx="2"/><rect x="7" y="4" width="10" height="14" rx="1" fill="#071523"/><circle cx="12" cy="20.5" r="1" fill="#071523"/></>;
  const net=<><path d="M5 2h12l3 3v11l-6 8H5z"/><rect x="8" y="6" width="9" height="7" rx="1" fill="#081525"/><path d="M11 12v4" stroke="#081525" strokeWidth="2"/></>;
  return <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">{name==='users'?users:name==='phone'?phone:name==='topology'?net:file}</svg>;
}

export function SkylineArt(){
  let seed=14;const rnd=()=>{seed=(seed*1664525+1013904223)>>>0;return seed/4294967296;};
  return <svg viewBox="0 0 225 185" preserveAspectRatio="none"><defs><linearGradient id="night-sky" x2="0" y2="1"><stop stopColor="#071322"/><stop offset=".64" stopColor="#343345"/><stop offset=".86" stopColor="#745244"/><stop offset="1" stopColor="#07101b"/></linearGradient></defs><rect width="225" height="185" fill="url(#night-sky)"/>{Array.from({length:29},(_,i)=>{const x=i*8-5,h=i===14?140:25+rnd()*69,y=174-h;return <g key={i}><path d={`M${x} 174V${y}h12V174Z`} fill={i%3?'#030b14':'#0b1420'}/>{i===14&&<path d={`M${x+6} ${y}v-24`} stroke="#637080" strokeWidth=".7"/>}{Array.from({length:Math.floor(h/7)},(_,j)=><g key={j}>{[2,6,10].map(k=><rect key={k} x={x+k} y={y+5+j*7} width="1.3" height="1.8" fill={rnd()>.48?'#f6b565':'#223342'} opacity={.4+rnd()*.5}/>)}</g>)}</g>})}<rect y="176" width="225" height="9" fill="#071421"/></svg>;
}
