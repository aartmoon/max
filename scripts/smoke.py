#!/usr/bin/env python3
"""Real REST + database smoke test; run against a running demo stack."""
import json, os, urllib.request, urllib.error
BASE=os.environ.get('API_URL','http://localhost:3000/api')
def call(path, body=None, status=200, method=None):
    data=None if body is None else json.dumps(body,ensure_ascii=False).encode()
    req=urllib.request.Request(BASE+path,data=data,headers={'Content-Type':'application/json'},method=method)
    try:
        response=urllib.request.urlopen(req,timeout=20)
    except urllib.error.HTTPError as e:
        response=e
    assert response.status==status,(path,response.status,response.read())
    return json.loads(response.read())
assert call('/health')['status']=='ok'
call('/requests',{'description':' ','address':'Москва'},400)
call('/requests/nope',status=400)
call('/requests/999999999',status=404)
for description,category,org in [('Течёт труба','PIPE_LEAK','1'),('Сломан лифт','ELEVATOR','2'),('Холодно в доме','HEATING','1'),('Мусор во дворе','OTHER','1')]:
    r=call('/requests',{'description':description,'address':'г. Москва, ул. Тестовая, д. 1'},201)
    assert (r['problemType'],r['responsibleOrganizationId'],r['status'])==(category,org,'CREATED')
    assert call('/requests/'+r['id'])['description']==description
    assert [h['status'] for h in call('/requests/'+r['id']+'/history')]==['CREATED']
    for target in ['SENT','ACCEPTED','IN_PROGRESS','RESOLVED']:
        assert call('/requests/'+r['id']+'/mock-next-status',{})['status']==target
    call('/requests/'+r['id']+'/mock-next-status',{},409)
    assert [h['status'] for h in call('/requests/'+r['id']+'/history')]==['CREATED','SENT','ACCEPTED','IN_PROGRESS','RESOLVED']
for kind in ['APPLICATION','QUESTION']:
    r=call('/requests',{'description':'Тестовая заявка или вопрос','address':'г. Москва, другой дом 20','kind':kind},201)
    assert r['kind']==kind
    assert call('/requests/'+r['id']+'/mock-next-status?reject=true',{})['status']=='REJECTED'
    call('/requests/'+r['id']+'/mock-next-status',{},409)
assert len(call('/requests'))>=6
print('PASS: health, validation, classification, routing, detail, list, status history, terminal states, applications/questions')

# The organization console and resident view share one persistent history.
for kind in ['EMERGENCY','COMPLAINT']:
    r=call('/requests',{'description':'Проверка кабинета УК','address':'г. Москва, ул. Тестовая, д. 1','kind':kind},201)
    path='/admin/requests/'+r['id']
    assert call(path)['kind']==kind
    call(path+'/status',{'status':'RESOLVED','comment':''},400,'PATCH')
    call(path+'/status',{'status':'REJECTED','comment':'  '},400,'PATCH')
    for target in ['ACCEPTED','IN_PROGRESS','RESOLVED']:
        updated=call(path+'/status',{'status':target,'comment':'Комментарий УК: '+target},method='PATCH')
        assert updated['status']==target
    history=call('/requests/'+r['id']+'/history')
    assert [h['status'] for h in history]==['CREATED','ACCEPTED','IN_PROGRESS','RESOLVED']
    assert history[-1]['comment']=='Комментарий УК: RESOLVED' and history[-1]['actor']=='УК · демо'
    call(path+'/status',{'status':'ACCEPTED'},409,'PATCH')
assert len(call('/admin/requests'))>=len(call('/requests'))
assert len(call('/organizations'))==2
print('PASS: new kinds, admin transitions, rejection validation, resident-visible comments')
