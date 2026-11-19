import test from 'node:test';
import assert from 'node:assert/strict';
const {default:register,relocateSystemPrompt,MARKER,SHORT_SYSTEM} = await import(process.env.AGY_GUARD_MODULE ?? new URL('./guard.mjs',import.meta.url).href);
const make = () => ({project:'aicode-consumers',model:'gemini-4-hydrogen-low',requestType:'agent',requestId:'test-id',request:{systemInstruction:{role:'user',parts:[{text:'Do not change protected files.\nConserva las reglas. 🔒'}]},contents:[{role:'user',parts:[{text:'hello'},{inlineData:{mimeType:'image/png',data:'test'}}]}],tools:[{functionDeclarations:[{name:'probe'}]}],labels:{request_id:'test'},generationConfig:{maxOutputTokens:100}}});
test('relocates complete instructions without mutating the input',()=>{
 const input=make(), snapshot=structuredClone(input), output=relocateSystemPrompt(input,'antigravity');
 assert.notStrictEqual(output,input);assert.deepEqual(input,snapshot);
 assert.equal(output.request.systemInstruction.parts[0].text,SHORT_SYSTEM);
 assert.equal(output.request.contents[0].parts[0].text,MARKER+input.request.systemInstruction.parts[0].text+'\n\n');
 assert.deepEqual(output.request.contents[0].parts.slice(1),input.request.contents[0].parts);
});
test('preserves generation metadata and tool declarations',()=>{
 const input=make(),out=relocateSystemPrompt(input,'antigravity');
 for(const key of ['project','model','requestType','requestId'])assert.deepEqual(out[key],input[key]);
 for(const key of ['tools','labels','generationConfig'])assert.deepEqual(out.request[key],input.request[key]);
});
test('preserves multiple instruction parts including Unicode',()=>{
 const input=make();input.request.systemInstruction.parts.push({text:'Segunda parte\n\t秘密'});
 const out=relocateSystemPrompt(input,'antigravity');
 assert.equal(out.request.contents[0].parts[0].text,MARKER+input.request.systemInstruction.parts.map(p=>p.text).join('\n\n')+'\n\n');
});
test('preserves tool-result and later turns',()=>{
 const input=make();input.request.contents=[{role:'user',parts:[{functionResponse:{name:'read',response:{text:'content'}}}]},{role:'model',parts:[{text:'done'}]}];
 const before=structuredClone(input.request.contents),out=relocateSystemPrompt(input,'antigravity');
 assert.deepEqual(out.request.contents[0].parts.slice(1),before[0].parts);assert.deepEqual(out.request.contents[1],before[1]);
});
test('inserts a user turn when no user turn exists',()=>{
 const input=make();input.request.contents=[{role:'model',parts:[{text:'prior'}]}];
 const out=relocateSystemPrompt(input,'antigravity');assert.equal(out.request.contents[0].role,'user');assert.deepEqual(out.request.contents.slice(1),input.request.contents);
});
test('handles an empty conversation',()=>{const input=make();input.request.contents=[];assert.equal(relocateSystemPrompt(input,'antigravity').request.contents.length,1);});
test('is idempotent',()=>{const once=relocateSystemPrompt(make(),'antigravity');assert.strictEqual(relocateSystemPrompt(once,'antigravity'),once);});
test('does not mistake a user marker for a previously applied guard',()=>{const input=make();input.request.contents[0].parts.unshift({text:MARKER+'user text'});assert.notStrictEqual(relocateSystemPrompt(input,'antigravity'),input);});
for(const provider of ['openai','anthropic',undefined])test('passes through foreign provider '+provider,()=>{const input=make();assert.strictEqual(relocateSystemPrompt(input,provider),input);});
test('passes through image generation',()=>{const input=make();input.requestType='image_gen';assert.strictEqual(relocateSystemPrompt(input,'antigravity'),input);});
test('supports explicit opt-out',()=>{const input=make();assert.strictEqual(relocateSystemPrompt(input,'antigravity',false),input);});
for(const bad of [null,[],{}, {requestType:'agent',request:null},{requestType:'agent',request:{systemInstruction:{parts:[{text:'text'},{inlineData:{}}]},contents:[]}}])test('passes through malformed or nontext input '+JSON.stringify(bad),()=>assert.strictEqual(relocateSystemPrompt(bad,'antigravity'),bad));
test('passes through blank system text',()=>{const input=make();input.request.systemInstruction.parts=[{text:' \n\t'}];assert.strictEqual(relocateSystemPrompt(input,'antigravity'),input);});
test('passes through malformed conversation turns',()=>{const input=make();input.request.contents=[null];assert.strictEqual(relocateSystemPrompt(input,'antigravity'),input);});
test('does not discard non-text system metadata',()=>{const input=make();input.request.systemInstruction.parts[0].extra='metadata';assert.strictEqual(relocateSystemPrompt(input,'antigravity'),input);});
test('registers a working Pi hook scoped to Antigravity',()=>{
 let hook;register({on:(name,callback)=>{assert.equal(name,'before_provider_request');hook=callback;}});
 const input=make();assert.equal(hook({payload:input},{model:{provider:'antigravity'}}).request.systemInstruction.parts[0].text,SHORT_SYSTEM);
 assert.equal(hook({payload:input},{model:{provider:'anthropic'}}),undefined);
});
