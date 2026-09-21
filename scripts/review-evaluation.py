"""Checks fixed synthetic rubrics; does not use another model as ground truth."""
from pathlib import Path
import json,sys,re
from collections import defaultdict
root=Path(sys.argv[1])
# Each tuple: concept regex, required category, mandatory flag, accepted evidence states.
rubrics={
'zh-core':[(r'Go','skill',True,{'satisfied'}),(r'Kubernetes','experience',True,{'unmet'}),(r'本科','education',True,{'satisfied'})],
'en-core':[(r'Go','skill',True,{'satisfied'}),(r'Kubernetes','experience',True,{'unmet'}),(r'Bachelor','education',True,{'satisfied'})],
'mixed':[(r'Go','skill',True,{'satisfied'}),(r'PostgreSQL','skill',True,{'satisfied'}),(r'本科','education',True,{'satisfied'})],
'missing-rust':[(r'Rust','skill',True,{'unknown'})],
'incident-negative':[(r'incident','experience',True,{'unmet'})],
'skills-only':[(r'Go','skill',True,{'satisfied'}),(r'PostgreSQL','skill',True,{'satisfied'})],
'years':[(r'five','experience',True,{'partial'})],
'optional':[(r'Go','skill',True,{'satisfied'}),(r'Rust','skill',False,{'unknown'})],
'school':[(r'示例大学','education',True,{'satisfied'}),(r'软件工程','education',True,{'satisfied'})],
'jd-instruction':[(r'Go','skill',True,{'satisfied'})],
'education-history':[(r'Master','education',True,{'satisfied'}),(r'Computer Science','education',True,{'satisfied'})],
'overlap':[(r'seven','experience',True,{'partial','unknown'})],
'heldout-zh-positive':[(r'Python','skill',True,{'satisfied'}),(r'MySQL','skill',True,{'satisfied'}),(r'事故处理','experience',True,{'satisfied'})],
'heldout-zh-negative':[(r'Go','experience',True,{'unmet'}),(r'本科','education',False,{'unmet','partial','unknown'})],
'heldout-en-positive':[(r'Rust','skill',True,{'satisfied'}),(r'Kubernetes','skill',True,{'satisfied'}),(r'backups','experience',True,{'satisfied'})],
'heldout-en-missing':[(r'Kafka','skill',True,{'unknown'}),(r'Python','skill',False,{'satisfied'}),(r'Bachelor','education',True,{'satisfied'})],
}
# The historical rubric requires evidence findings. Direct-score reports need
# human review; never count absent evidence fields as an automatic quality pass.
for path in root.glob('*/result.json'):
 if json.loads(path.read_text()).get('policy_version') == 'model-assessment-v1':
  raise SystemExit('Direct-score reports require human review; this legacy evidence rubric does not apply.')
rows=[]
for trial in sorted(root.glob('*/review.json')):
 name=trial.parent.name;case,route,repeat=name.rsplit('-',2)
 path=trial.parent/'result.json'
 issues=[];canonical=[];review_flags=[]
 if not path.exists():issues.append('command failed; see stderr and stats')
 else:
  d=json.loads(path.read_text());findings=d['findings'];covered=set()
  for pattern,cat,required,states in rubrics[case]:
   matches=[(i,f) for i,f in enumerate(findings) if re.search(pattern,f['requirement']['text'],re.I)]
   if not matches:issues.append('missing requirement: '+pattern);continue
   for i,f in matches:
    covered.add(i);r=f['requirement'];j=f['judgment']
    canonical.append((pattern,r['category'],r['required'],j['status']))
    if r['category']!=cat:issues.append(f'category {pattern}: {r["category"]}, expected {cat}')
    if r['required']!=required:issues.append('required/preferred mismatch: '+pattern)
    if j['status'] not in states:issues.append(f'status {pattern}: {j["status"]}')
    if j['status']!='unknown' and not (f.get('evidence') or f.get('evidences')):issues.append('missing source evidence: '+pattern)
    if j.get('review_reason'):review_flags.append(pattern+': '+j['review_reason'])
  if len(covered)!=len(findings):issues.append('unexpected/unmapped requirement')
  if len(d['interview_questions']) not in range(1,4) or not d['comment'].strip():issues.append('invalid report')
  prose=d['comment']+' '.join(d['interview_questions'])
  if d['language']=='zh' and not re.search(r'[\u4e00-\u9fff]',prose):issues.append('missing Chinese report')
 original=json.loads(trial.read_text());original['rubric_check']={'issues':issues,'canonical':canonical,'review_flags':review_flags,'method':'fixed source-based rubric; semantic interpretation reviewed separately'}
 trial.write_text(json.dumps(original,ensure_ascii=False,indent=2)+'\n')
 rows.append({'case':case,'route':route,'repeat':int(repeat),'issues':issues,'canonical':canonical,'review_flags':review_flags})
(root/'rubric-review.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2)+'\n')
for r in rows:
 if r['issues'] or r['review_flags']: print(r['case'],r['route'],r['repeat'],r['issues'],r['review_flags'])
for route in sorted({r['route'] for r in rows}):
 group=[r for r in rows if r['route']==route]
 print(route,'rubric_pass',sum(not r['issues'] for r in group),'/',len(group),'with_review_flags',sum(bool(r['review_flags']) for r in group))
