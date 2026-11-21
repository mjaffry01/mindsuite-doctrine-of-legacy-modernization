/* MindSuite web app · icons for the HLD diagrams.
   Original line icons on a 24×24 grid, drawn white on a coloured badge in the corner of each box.
   The icon is chosen by component id, else by words in the title (so boxes drawn by the Diagram agent get
   one too); the badge colour follows the zone (edge, application, data, network, cross-cutting, external).
   Generic shapes only: no vendor logos, so the diagram stays provider-neutral and safe to share. */

const HLD_ICON_PATHS={
 user:'<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 3.6-6 8-6s8 2 8 6"/>',
 users:'<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20c0-3.5 3-5.5 6.5-5.5s6.5 2 6.5 5.5"/><path d="M15.5 4.8a3.5 3.5 0 0 1 0 6.4M18 14.8c2.2.6 3.5 2.4 3.5 5.2"/>',
 staff:'<circle cx="12" cy="7" r="3.6"/><path d="M4.5 21c0-3.8 3.3-6 7.5-6s7.5 2.2 7.5 6"/><path d="M12 15l-1.4 2.4L12 21l1.4-3.6z"/>',
 device:'<rect x="7" y="7" width="10" height="10" rx="1.5"/><path d="M9.5 3v4M14.5 3v4M9.5 17v4M14.5 17v4M3 9.5h4M3 14.5h4M17 9.5h4M17 14.5h4"/>',
 channels:'<rect x="2.5" y="4" width="13" height="10" rx="1.5"/><path d="M6 18h6M9 14v4"/><rect x="16.5" y="8.5" width="5" height="11.5" rx="1"/>',
 globe:'<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3c2.5 2.6 3.6 5.6 3.6 9s-1.1 6.4-3.6 9c-2.5-2.6-3.6-5.6-3.6-9S9.5 5.6 12 3z"/>',
 cdn:'<path d="M7 18.5a4.3 4.3 0 0 1-.6-8.6A6 6 0 0 1 18 9.2a4.6 4.6 0 0 1-.4 9.3z"/><path d="M12 10.5v5.5M9.6 13.8l2.4 2.4 2.4-2.4"/>',
 shield:'<path d="M12 2.8l8 3v6c0 5-3.4 8.2-8 9.4-4.6-1.2-8-4.4-8-9.4v-6z"/><path d="M8.6 12.2l2.4 2.4 4.4-4.6"/>',
 balancer:'<circle cx="12" cy="5" r="2.4"/><circle cx="4.8" cy="19" r="2.4"/><circle cx="12" cy="19" r="2.4"/><circle cx="19.2" cy="19" r="2.4"/><path d="M12 7.4v9.2M11 9l-5.2 7.8M13 9l5.2 7.8"/>',
 api:'<path d="M8 6l-5 6 5 6M16 6l5 6-5 6"/><path d="M13.6 4l-3.2 16"/>',
 app:'<path d="M12 2.5l8.5 4.8v9.4L12 21.5l-8.5-4.8V7.3z"/><path d="M3.5 7.3L12 12l8.5-4.7M12 12v9.5"/>',
 server:'<rect x="3" y="3.5" width="18" height="7.5" rx="1.5"/><rect x="3" y="13" width="18" height="7.5" rx="1.5"/><path d="M7 7.2h2M7 16.7h2"/>',
 bolt:'<path d="M13 2L4.5 13.5H11l-1 8.5 8.5-11.5H12z"/>',
 gear:'<circle cx="12" cy="12" r="3.2"/><path d="M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.3 5.3l2.1 2.1M16.6 16.6l2.1 2.1M5.3 18.7l2.1-2.1M16.6 7.4l2.1-2.1"/>',
 clock:'<circle cx="12" cy="12" r="9"/><path d="M12 7v5.2l3.4 2.1"/>',
 database:'<ellipse cx="12" cy="5.5" rx="8" ry="3"/><path d="M4 5.5v13c0 1.7 3.6 3 8 3s8-1.3 8-3v-13"/><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/>',
 replica:'<ellipse cx="10" cy="5.5" rx="6.5" ry="2.5"/><path d="M3.5 5.5v11c0 1.4 2.9 2.5 6.5 2.5"/><path d="M16.5 5.5V10"/><path d="M14 14.5h7M18.5 12l2.5 2.5-2.5 2.5"/>',
 cache:'<rect x="3" y="6.5" width="18" height="11" rx="1.5"/><path d="M7 6.5v11M11 6.5v11M15 6.5v11M5.5 17.5v3M18.5 17.5v3M5.5 3.5v3M18.5 3.5v3"/>',
 queue:'<rect x="2.5" y="8" width="5" height="8" rx="1"/><rect x="9.5" y="8" width="5" height="8" rx="1"/><rect x="16.5" y="8" width="5" height="8" rx="1"/><path d="M2.5 20h19M18 18l2 2-2 2"/>',
 stream:'<path d="M3 7.5c3-3 6 3 9 0s6 3 9 0M3 12.5c3-3 6 3 9 0s6 3 9 0M3 17.5c3-3 6 3 9 0s6 3 9 0"/>',
 search:'<circle cx="10.5" cy="10.5" r="6.5"/><path d="M15.5 15.5l5.5 5.5"/>',
 bucket:'<ellipse cx="12" cy="6" rx="8" ry="2.6"/><path d="M4 6l2 13.2c.2 1.3 2.8 2.3 6 2.3s5.8-1 6-2.3L20 6"/>',
 chart:'<path d="M3 21h18"/><path d="M6 17v-6M11 17V5M16 17v-8M20.5 17v-3"/>',
 vault:'<rect x="3" y="3.5" width="18" height="17" rx="2"/><circle cx="12" cy="12" r="4"/><path d="M12 8v1.5M12 14.5V16M8 12h1.5M14.5 12H16M3 8h-1M3 16h-1"/>',
 network:'<rect x="9" y="2.5" width="6" height="5" rx="1"/><rect x="2.5" y="16.5" width="6" height="5" rx="1"/><rect x="15.5" y="16.5" width="6" height="5" rx="1"/><path d="M12 7.5V12M5.5 16.5V12h13v4.5"/>',
 outbound:'<path d="M14 3.5h6.5V10M20.5 3.5L11 13"/><path d="M18 14v5.5a1 1 0 0 1-1 1H4.5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1H10"/>',
 terminal:'<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 9l3 3-3 3M12.5 15H17"/>',
 link:'<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
 idcard:'<rect x="2.5" y="5" width="19" height="14" rx="2"/><circle cx="8.5" cy="11" r="2.3"/><path d="M5 16.2c.6-1.6 2-2.4 3.5-2.4s2.9.8 3.5 2.4M14.5 10h4.5M14.5 13.5h4.5"/>',
 key:'<circle cx="7.5" cy="15.5" r="4"/><path d="M10.4 12.6L20 3M16.2 6.8l2.8 2.8M13.8 9.2l2 2"/>',
 sliders:'<path d="M4 6h9.5M18.5 6H20M4 12h3.5M12.5 12H20M4 18h11.5M20.5 18H20"/><circle cx="16" cy="6" r="2.2"/><circle cx="10" cy="12" r="2.2"/><circle cx="18" cy="18" r="2.2"/>',
 pulse:'<path d="M2.5 12h4l3-7 4.5 14 3-7h4.5"/>',
 logs:'<path d="M6 2.5h9l4 4V21.5H6z"/><path d="M15 2.5v4h4M9 11h7M9 14.5h7M9 18h4.5"/>',
 trace:'<circle cx="5" cy="5.5" r="2.2"/><circle cx="19" cy="12" r="2.2"/><circle cx="5" cy="18.5" r="2.2"/><path d="M7.2 5.5H13a3.2 3.2 0 0 1 0 6.5h3.8M16.8 12H11a3.2 3.2 0 0 0 0 6.5H7.2"/>',
 dashboard:'<rect x="3" y="3" width="8" height="9" rx="1.5"/><rect x="13" y="3" width="8" height="5" rx="1.5"/><rect x="13" y="10" width="8" height="11" rx="1.5"/><rect x="3" y="14" width="8" height="7" rx="1.5"/>',
 audit:'<rect x="5" y="4" width="14" height="17.5" rx="2"/><path d="M9 4V2.5h6V4M8.8 13l2.2 2.2 4.2-4.4"/>',
 eye:'<path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
 registry:'<path d="M12 2.5l9 4.5-9 4.5L3 7z"/><path d="M3 12l9 4.5 9-4.5M3 16.8l9 4.5 9-4.5"/>',
 cicd:'<path d="M20 10.5A8 8 0 0 0 5.6 6.4L3 9M4 13.5a8 8 0 0 0 14.4 4.1L21 15"/><path d="M3 3.8V9h5.2M21 20.2V15h-5.2"/>',
 policy:'<path d="M5 3.5h10.5l3.5 3.5v14.5H5z"/><path d="M8.5 9h7M8.5 12.5h7M8.5 16h4"/>',
 dr:'<rect x="2.5" y="8.5" width="7" height="12" rx="1"/><rect x="14.5" y="8.5" width="7" height="12" rx="1"/><path d="M5 12h2M5 15.5h2M17 12h2M17 15.5h2M9 5h6.5M13.5 3l2 2-2 2"/>',
 card:'<rect x="2.5" y="5" width="19" height="14" rx="2"/><path d="M2.5 9.8h19M6 15h4.5"/>',
 mail:'<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3.5 6.5L12 13l8.5-6.5"/>',
 cloud:'<path d="M7 18.5a4.5 4.5 0 0 1-.6-9A6 6 0 0 1 18 8.8a4.8 4.8 0 0 1-.4 9.7z"/>',
 partner:'<circle cx="8.8" cy="12" r="5.3"/><circle cx="15.2" cy="12" r="5.3"/>',
 ai:'<path d="M11 3l1.9 5.1L18 10l-5.1 1.9L11 17l-1.9-5.1L4 10l5.1-1.9z"/><path d="M18.5 14.5l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8z"/>',
 systems:'<rect x="2.5" y="3" width="8" height="7" rx="1"/><rect x="13.5" y="14" width="8" height="7" rx="1"/><path d="M6.5 10v4.3a2 2 0 0 0 2 2h5M17.5 14V9.7a2 2 0 0 0-2-2h-5"/>',
 broadcast:'<circle cx="12" cy="12" r="2"/><path d="M8.3 8.3a5.3 5.3 0 0 0 0 7.4M15.7 8.3a5.3 5.3 0 0 1 0 7.4M5.4 5.4a9.4 9.4 0 0 0 0 13.2M18.6 5.4a9.4 9.4 0 0 1 0 13.2"/>',
 funnel:'<path d="M3 4h18l-7 8.2v6.8l-4 2v-8.8z"/>',
 video:'<rect x="2.5" y="5.5" width="14" height="13" rx="2"/><path d="M16.5 10.2l5-3v9.6l-5-3"/>',
 module:'<rect x="3.5" y="3.5" width="7" height="7" rx="1.5"/><rect x="13.5" y="3.5" width="7" height="7" rx="1.5"/><rect x="3.5" y="13.5" width="7" height="7" rx="1.5"/><rect x="13.5" y="13.5" width="7" height="7" rx="1.5"/>',
 platform:'<rect x="3" y="3.5" width="18" height="6" rx="1.5"/><rect x="3" y="14.5" width="18" height="6" rx="1.5"/><path d="M8 9.5v5M16 9.5v5"/>'
};

// Component id → icon (ids from hldComponents()).
const HLD_ICON_BY_ID={customers:'users',users:'users',staff:'staff',devices:'device',dns:'globe',cdn:'cdn',waf:'shield',lb:'balancer',apigw:'api',
 iothub:'broadcast',ingest:'funnel',app:'app',workers:'gear',jobs:'clock',fn:'bolt',db:'database',dbreplica:'replica',cache:'cache',mq:'queue',
 events:'stream',stream:'stream',search:'search',object:'bucket',media:'video',tsdb:'chart',analytics:'chart',backupvault:'vault',privep:'network',
 nat:'outbound',bastion:'terminal',vpn:'link',idstaff:'idcard',idcust:'idcard',vault:'key',appconfig:'sliders',metrics:'pulse',logs:'logs',
 apm:'trace',dash:'dashboard',audit:'audit',siem:'eye',registry:'registry',cicd:'cicd',policy:'policy',drsite:'dr',pay:'card',notify:'mail',
 vendor:'cloud',partner:'partner',ai:'ai',other:'systems'};

// Words in the title → icon, for boxes without a known id (Diagram agent, architecture diagram).
const HLD_ICON_WORDS=[
 [/payment|checkout|billing/i,'card'],[/email|sms|notif/i,'mail'],[/partner platform/i,'partner'],[/other systems?/i,'systems'],[/identity|sign.?in/i,'idcard'],
 [/customer|viewer|subscriber|shopper|users?\b|people/i,'users'],[/staff|admin|editor|operator|agent\b|manager/i,'staff'],[/device|sensor|iot\b/i,'device'],
 [/\bdns\b|domain/i,'globe'],[/cdn|front door|edge cache|content delivery|cloudfront/i,'cdn'],[/waf|firewall|ddos|shield/i,'shield'],
 [/load balanc|traffic manager/i,'balancer'],[/api|gateway/i,'api'],[/device hub|iot hub|mqtt/i,'broadcast'],[/ingest/i,'funnel'],
 [/function|serverless|lambda/i,'bolt'],[/worker|background/i,'gear'],[/schedul|job|cron|batch/i,'clock'],
 [/replica/i,'replica'],[/database|\bdb\b|sql|postgres|mysql|oracle|cosmos|dynamo|firestore|mongo/i,'database'],[/cache|redis|memcache/i,'cache'],
 [/queue|service bus|sqs/i,'queue'],[/messag|event|stream|kafka|pub\/?sub|kinesis/i,'stream'],[/search|opensearch|elastic/i,'search'],
 [/file|media|object|blob|storage|bucket|s3\b/i,'bucket'],[/video|transcod|stream(ing)? video|playback/i,'video'],[/analytic|warehouse|report|time.?series/i,'chart'],
 [/backup/i,'vault'],[/private (endpoint|link)|subnet|virtual network|vnet|vpc/i,'network'],[/\bnat\b|egress|outbound/i,'outbound'],[/bastion|jump/i,'terminal'],
 [/vpn|direct connect|expressroute|interconnect|site.to.site/i,'link'],[/sign.?in|identity|login|entra|cognito|sso|auth/i,'idcard'],
 [/secret|key vault|\bkeys?\b|kms|certificate/i,'key'],[/setting|config|feature flag/i,'sliders'],[/metric|alert|monitor/i,'pulse'],[/\blogs?\b|logging/i,'logs'],
 [/trac|apm|performance/i,'trace'],[/dashboard|on.?call/i,'dashboard'],[/audit/i,'audit'],[/siem|security monitor|sentinel/i,'eye'],
 [/registry|image/i,'registry'],[/ci\/?cd|pipeline|deploy|release/i,'cicd'],[/policy|governance|guardrail/i,'policy'],[/disaster|recovery|second region|failover/i,'dr'],
 [/payment|checkout|billing|card/i,'card'],[/email|sms|notif|push/i,'mail'],[/saas|bought|vendor/i,'cloud'],[/partner/i,'partner'],
 [/\bai\b|model|assistant|bedrock|openai|vertex/i,'ai'],[/other system|integration|external system|erp|crm|accounting/i,'systems'],
 [/web|mobile|channel|app front/i,'channels'],[/platform|container|kubernetes|app service|ecs|cloud run|fargate|application|backend|service/i,'app']];

// Badge colour per zone / box class.
const HLD_ICON_COLOR={actors:'#55657f',actor:'#55657f',edge:'#2f6fc0',app:'#5b46cf',module:'#5b46cf',data:'#23895b',net:'#5f6e8d',
 cross:'#7b4fcc',ext:'#b97812',id:'#7b4fcc',pay:'#c2484f',platform:'#5b46cf'};

function hldIconKey(id,title,sub){
 if(id&&HLD_ICON_BY_ID[id])return HLD_ICON_BY_ID[id];
 const t=String(title||'');for(const [re,k] of HLD_ICON_WORDS)if(re.test(t))return k;
 const s=String(sub||'');for(const [re,k] of HLD_ICON_WORDS)if(re.test(s))return k;
 return 'app';
}
function hldIconColor(cls){const parts=String(cls||'').split(/\s+/).reverse();for(const p of parts)if(HLD_ICON_COLOR[p])return HLD_ICON_COLOR[p];return '#5b6478';}
// A badge with the icon, top-left at (x,y), size s (px).
function hldIconBadge(key,x,y,s,color){
 const p=HLD_ICON_PATHS[key]||HLD_ICON_PATHS.app,k=(s*0.68)/24,o=s*0.16;
 return '<g class="hic" aria-hidden="true"><rect x="'+x+'" y="'+y+'" width="'+s+'" height="'+s+'" rx="'+(s*0.27)+'" style="fill:'+color+';stroke:none"/>'+
  '<g class="ig" transform="translate('+(x+o)+' '+(y+o)+') scale('+k.toFixed(4)+')" fill="none" stroke="#fff" stroke-width="2.1" stroke-linecap="round" stroke-linejoin="round">'+p+'</g></g>';
}
// The diagrams style every rect in a box; these rules keep the badge colour and the white icon lines.
const HLD_ICON_CSS='.hic .ig *{fill:none!important;stroke:#fff!important}';
