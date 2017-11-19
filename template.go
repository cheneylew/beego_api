
package main

var MY_TPL=`
//
//  APStockManager.m
//  App
//
//  Created by Dejun Liu on 2017/10/23.
//  Copyright © 2017年 Dejun Liu. All rights reserved.
//

#import "APStockManager.h"
#import <AFNetworking/AFNetworking.h>
#import <NavyUIKit/NSString+Category.h>

#define INDEX_MIN 0
#define INDEX_MAX 5000

@interface APStockManager ()

PP_STRONG(AFURLSessionManager, manager);

@property (nonatomic, strong)   NSMutableArray<APStock *> *(stocks);
PP_ASSIGN_BASIC(NSInteger, stocksQueryCount);
PP_ASSIGN_BASIC(NSInteger, stocksQueryTotalCount);

PP_BOOL(downloadRetry);
PP_STRING(downloadCodes);
@end

@implementation APStockManager

SINGLETON_IMPL(APStockManager)

- (instancetype)init
{
    self = [super init];
    if (self) {
        self.stocks = [NSMutableArray array];
    }
    return self;
}

#pragma mark -公共方法
+ (NSMutableArray<APStockItem *> *)parseStockItemsWithSinaHtml:(NSString *) dataString {
    if (dataString == nil && dataString.length <= 0) {
        return nil;
    }
    
    dataString = [dataString jk_sinaStockHistoryTableHtml];
    
    WEAK_SELF;
    NSData  * data      = [dataString dataUsingEncoding:NSUTF8StringEncoding];
    TFHpple * doc       = [[TFHpple alloc] initWithHTMLData:data];
    NSArray<TFHppleElement *> * elements  = [doc searchWithXPathQuery:@"//table[@id='FundHoldSharesTable']"];
    
    NSMutableArray<APStockItem *> *items = [NSMutableArray array];
    [elements enumerateObjectsUsingBlock:^(TFHppleElement * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
        NSArray<TFHppleElement *> *rows = [obj childrenWithTagName:@"tr"];
        [rows enumerateObjectsUsingBlock:^(TFHppleElement * _Nonnull row, NSUInteger idx, BOOL * _Nonnull stop) {
            if (idx > 0) {
                NSArray<TFHppleElement *> *cols = [row childrenWithTagName:@"td"];
                APStockItem *item = [APStockItem new];
                NSString *date = [APStockManager stockItemPropertyWithInElement:cols[0]];
                item.date = [APStockManager dateWithString:date
                                              format:@"yyyy-MM-dd"];
                item.startPrice = [[APStockManager stockItemPropertyWithInElement:cols[1]] doubleValue];
                item.maxPrice = [[APStockManager stockItemPropertyWithInElement:cols[2]] doubleValue];
                item.endPrice = [[APStockManager stockItemPropertyWithInElement:cols[3]] doubleValue];
                item.minPrice = [[APStockManager stockItemPropertyWithInElement:cols[4]] doubleValue];
                item.dealNum = [[APStockManager stockItemPropertyWithInElement:cols[5]] integerValue];
                item.dealMoney = [[APStockManager stockItemPropertyWithInElement:cols[6]] integerValue];
                [items addObject:item];
            }
        }];
    }];
    
    return items;
}

+ (NSString *)stockItemPropertyWithInElement:(TFHppleElement *) element {
    return [[[[[[element firstChildWithTagName:@"div"].raw jk_stringByRemovingScriptsAndStrippingHTML] removeWhiteSpace] stringByReplacingOccurrencesOfString:@"&#13;" withString:@""] removeWhiteSpace] jk_trimmingWhitespaceAndNewlines];
}

+ (NSDate *)getDateWithString:(NSString *) dateString {
    NSDateFormatter *dateFormat = [[NSDateFormatter alloc] init];
    [dateFormat setDateFormat:@"dd/MM/yyyy"];
    NSDate *date = [dateFormat dateFromString:dateString];
    return date;
}

+ (NSDate *)dateWithString:(NSString *) dateString format:(NSString *) format {
    NSDateFormatter *dateFormat = [[NSDateFormatter alloc] init];
    [dateFormat setDateFormat:format];
    NSDate *date = [dateFormat dateFromString:dateString];
    return date;
}

#pragma mark -私有方法

- (AFURLSessionManager *)manager {
    if (_manager == nil) {
        AFHTTPResponseSerializer *responseSerializer = [AFHTTPResponseSerializer serializer];
        
        NSURLSessionConfiguration *configuration = [NSURLSessionConfiguration defaultSessionConfiguration];
        configuration.HTTPMaximumConnectionsPerHost = 3;
        AFURLSessionManager *manager = [[AFURLSessionManager alloc] initWithSessionConfiguration:configuration];
        manager.securityPolicy = [AFSecurityPolicy policyWithPinningMode:AFSSLPinningModeNone];
        manager.securityPolicy.allowInvalidCertificates = YES;
        manager.responseSerializer = responseSerializer;
        [manager.securityPolicy setValidatesDomainName:NO];
        
        _manager = manager;
    }
    
    return _manager;
}

- (void)parseStockCode:(NSString *) codes {
    self.downloadCodes = codes;
    [PDHttpClient showLoadingInView:[UIApplication sharedApplication].keyWindow title:@"下载股票信息..."];
    WEAK_SELF;
    dispatch_async(dispatch_get_global_queue(0, 0), ^{
        NSRegularExpression *regex = [NSRegularExpression regularExpressionWithPattern:@"\\d{6}" options:0 error:NULL];
        NSArray* array = [regex matchesInString:codes options:0 range:NSMakeRange(0, [codes length])];
        NSMutableArray<NSString *>* stringArray = [[NSMutableArray alloc] init];
        //当解析出的数组至少有一个对象时，即原文本中存在至少一个符合规则的字段
        if (array.count != 0) {
            for (NSTextCheckingResult* result in array) {
                //从NSTextCheckingResult类中取出range属性
                NSRange range = result.range;
                //从原文本中将字段取出并存入一个NSMutableArray中
                [stringArray addObject:[codes substringWithRange:range]];
            }
        }
        
        
        DLog(@"共%ld只股票", stringArray.count);
        weakself.stocksQueryTotalCount = 0;
        weakself.stocksQueryCount = 0;
        [stringArray enumerateObjectsUsingBlock:^(NSString * _Nonnull stockCode, NSUInteger idx, BOOL * _Nonnull stop) {
            if (idx >=INDEX_MIN && idx < INDEX_MAX) {
                //有缓存读缓存
                RLMResults<APStock *> *stocks = [APStock objectsWhere:[NSString stringWithFormat:@"code = '%@'", stockCode]];
                if (stocks.count) {
                    //增量更新
                    APStock *stock = stocks.firstObject;
                    NSString *lastStr = [stock.lastDate jk_stringWithFormat:@"dd/MM/yyyy"];
                    NSString *nowStr = [[NSDate date] jk_stringWithFormat:@"dd/MM/yyyy"];
                    
                    NSDate *lastDate = [APStockManager getDateWithString:lastStr];
                    NSDate *nowDate = [APStockManager getDateWithString:nowStr];
                    NSInteger days = [lastDate jk_daysBeforeDate:nowDate];
                    if (days) {
                        weakself.stocksQueryTotalCount ++;
                        [weakself queryStockWithNum:stockCode days:(days + 1) completion:^(NSError *error, NSString *result) {
                            if (!error) {
                                [weakself parseOneItem:result stockCode:stockCode  addUpdate:YES];
                            } else {
                                DLog(@"request error: %@", error);
                                weakself.downloadRetry = YES;
                                [PDHttpClient showErrorToast:error.localizedDescription afterDelay:1.5];
                            }
                            
                            weakself.stocksQueryCount ++;
                            if (weakself.stocksQueryCount == weakself.stocksQueryTotalCount) {
                                [weakself downloadFinished];
                            }
                        }];
                    }
                    DLog(@"");
                    return ;
                } else {
                    //没有缓存查询
                    weakself.stocksQueryTotalCount ++;
                    [weakself queryStockWithNum:stockCode days:90 completion:^(NSError *error, NSString *result) {
                        if (!error) {
                            [weakself parseOneItem:result stockCode:stockCode addUpdate:NO];
                        } else {
                            DLog(@"request error: %@", error);
                            weakself.downloadRetry = YES;
                            [PDHttpClient showErrorToast:error.localizedDescription afterDelay:1.5];
                        }
                        
                        weakself.stocksQueryCount ++;
                        if (weakself.stocksQueryCount == weakself.stocksQueryTotalCount) {
                            [weakself downloadFinished];
                        }
                    }];
                }
            }
        }];
    });
    
    
}

- (void)downloadFinished {
    //根据error判断是否需要重试
    if (self.downloadRetry) {
        [PDHttpClient hideLoadingInView:[UIApplication sharedApplication].keyWindow];
        [PDHttpClient showLoadingInView:[UIApplication sharedApplication].keyWindow title:@"20秒钟后重试..."];
        self.downloadRetry = NO;
        [self jk_performBlock:^{
            [self parseStockCode:self.downloadCodes];
        } afterDelay:20];
    } else {
        [PDHttpClient hideLoadingInView:[UIApplication sharedApplication].keyWindow];
        [UIAlertView jk_alertWithCallBackBlock:^(NSInteger buttonIndex) {
        } title:@"" message:@"所有数据同步完成" cancelButtonName:@"确认" otherButtonTitles:nil];
    }
}

- (void)parseOneItem:(NSString *) itemContent stockCode:(NSString *) code addUpdate:(BOOL) isAddUpdate {
    NSMutableArray<APStockItem *> *items = [NSMutableArray array];
    NSArray<NSString *> *rowStrs = [itemContent componentsSeparatedByString:@"\n"];
    [rowStrs enumerateObjectsUsingBlock:^(NSString * _Nonnull itemStr, NSUInteger idx, BOOL * _Nonnull stop) {
        if (idx >= 8) {
            NSArray *slices = [itemStr componentsSeparatedByString:@","];
            if (slices.count == 6) {
                APStockItem *item = [APStockItem new];
                item.startPrice = [slices[1] doubleValue];
                item.maxPrice = [slices[2] doubleValue];
                item.minPrice = [slices[3] doubleValue];
                item.endPrice = [slices[4] doubleValue];
                item.dealNum = [slices[5] doubleValue];
                
                [items addObject:item];
            }
        }
    }];
    
    __block NSInteger increaseCount = 0;
    __block NSInteger decreaseCount = 0;
    [items enumerateObjectsUsingBlock:^(APStockItem * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
        NSInteger curIdx = idx - 1;
        if (curIdx >= 0) {
            double delta = (obj.midPrice - items[curIdx].midPrice);
//            DLog(@"%.2f", delta);
            if (delta >= 0) {
                increaseCount ++;
            } else {
                decreaseCount ++;
            }
        }
    }];
    
    DLog(@"代码：%@ 增长次数：%ld 减少次数：%ld", code, increaseCount, decreaseCount)
    APStock *stock = nil;
    if (isAddUpdate) {
        RLMResults<APStock *> *stocks = [APStock objectsWhere:[NSString stringWithFormat:@"code = '%@'", code]];
        if (stocks.count <= 0) {//不应该出现这种情况
            ASSERT_NIL(nil);
            return;
        }
        APStock *firstStock = stocks.firstObject;
        stock = firstStock;
        
        //更新DB
        RLMRealm *realm = [RLMRealm defaultRealm];
        [realm transactionWithBlock:^{
            stock.lastDate      = [NSDate date];
            [items enumerateObjectsUsingBlock:^(APStockItem * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
                [stock.stockItems addObject:obj];
            }];
        }];
        
    } else {
        stock = [APStock new];
        [items enumerateObjectsUsingBlock:^(APStockItem * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
            [stock.stockItems addObject:obj];
        }];
        stock.code = code;
        stock.increaseTimes = increaseCount;
        stock.decreaseTimes = decreaseCount;
        stock.lastDate      = [NSDate date];
        
        //添加
        if (stock.increaseTimes == 0 && stock.decreaseTimes == 0) {
            
        } else {
            // Persist your data easily
            RLMRealm *realm = [RLMRealm defaultRealm];
            [realm transactionWithBlock:^{
                [realm addObject:stock];
            }];
        }
    }
    
    @synchronized(self) {
        if (!stock) {
            return;
        }
        [self.stocks addObject:stock];
    }
}

- (void)queryStockWithNum:(NSString *)stockNum days:(NSInteger)days completion:(StockHtmlCompletion) completion{
    NSString *urlString = [NSString stringWithFormat:@"https://finance.google.com.hk/finance/getprices?q=%@&p=%ldd",
                           stockNum,
                           days];
    DLog(@"%@", urlString);
    NSURL *URL = [NSURL URLWithString:urlString];
    NSURLRequest *request = [NSURLRequest requestWithURL:URL];
    
    NSURLSessionDataTask *dataTask = [self.manager dataTaskWithRequest:request completionHandler:^(NSURLResponse *response, id responseObject, NSError *error) {
        if (error) {
            BLOCK_SAFE_RUN(completion, error, nil);
        } else {
            NSString *responseStr = [NSString stringWithUTF8String:[responseObject bytes]];
            BLOCK_SAFE_RUN(completion, nil, responseStr);
        }
    }];
    [dataTask resume];
}

- (void)startAnilysisLocalWithDealDays:(NSInteger) dealDays {
    [PDHttpClient showLoadingInView:[UIApplication sharedApplication].keyWindow title:@"分析中"];
    
    dispatch_async(dispatch_get_global_queue(0, 0), ^{
        NSMutableArray<APStock *> *stocks = [NSMutableArray array];
        RLMResults<APStock *> *allItems = [APStock allObjects];
        for (NSInteger i = 0; i < allItems.count; i++) {
            APStock *stock = [allItems objectAtIndex:i];
            stock.calDecreaseTimes = [stock calDecreaseTimesWithDealDays:dealDays];
            stock.calIncreaseTimes = [stock calIncreaseTimesWithDealDays:dealDays];
            stock.calIncreaseRate = [stock calIncreaseRateWithDealDays:dealDays];
            [stocks addObject: stock];
        }
        
        [stocks sortUsingComparator:^NSComparisonResult(APStock *obj1, APStock *obj2) {
            if (obj1.calIncreaseRate > obj2.calIncreaseRate) {
                return NSOrderedAscending;
            } else if (obj1.calIncreaseRate == obj2.calIncreaseRate) {
                return NSOrderedSame;
            } else {
                return NSOrderedDescending;
            }
        }];
        
        __block NSString *result = [NSString stringWithFormat:@"最近%ld天 共%ld只", dealDays, stocks.count];
        [stocks enumerateObjectsUsingBlock:^(APStock * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
            if (idx < 500) {
                if ([obj.code hasPrefix:@"30"] ||
                    [obj.code hasPrefix:@"002"] ||
                    [obj.code hasPrefix:@"60"] ||
                    [obj.code hasPrefix:@"900"] ||
                    [obj.code hasPrefix:@"20"] ||
                    [obj.code hasPrefix:@"00"] ) {
                    NSInteger inTimes = [obj calIncreaseTimes];
                    NSInteger deTimes = [obj calDecreaseTimes];
                    NSString *str = [NSString stringWithFormat:@"%@\tIN:%ld\tDE:%ld\tRT:%.2f", obj.code, inTimes, deTimes,[obj calIncreaseRateWithDealDays:dealDays]];
                    result = [NSString stringWithFormat:@"%@\n%@",result,str];
                }
            }
        }];
        
        dispatch_async(dispatch_get_main_queue(), ^{
            [PDHttpClient hideLoadingInView:[UIApplication sharedApplication].keyWindow];
            [UIAlertView jk_alertWithCallBackBlock:^(NSInteger buttonIndex) {
                //
            } title:@"" message:result cancelButtonName:@"ok" otherButtonTitles: nil];
        });
    });
    
}

#pragma mark 新浪历史数据查询
- (void)querySinaStockWithNum:(NSString *)stockNum
                         year:(NSString *)year
                      quarter:(NSString *)quarter
                   completion:(StockHtmlCompletion) completion{
    NSString *urlString = [NSString stringWithFormat:@"http://money.finance.sina.com.cn/corp/go.php/vMS_MarketHistory/stockid/%@.phtml?year=%@&jidu=%@",
                           stockNum,
                           year,
                           quarter];
    if (year.length == 0 || quarter.length == 0) {
        urlString = [NSString stringWithFormat:@"http://money.finance.sina.com.cn/corp/go.php/vMS_MarketHistory/stockid/%@.phtml",
                     stockNum];
    }
    DLog(@"%@", urlString);
    NSURL *URL = [NSURL URLWithString:urlString];
    NSURLRequest *request = [NSURLRequest requestWithURL:URL];
    
    NSURLSessionDataTask *dataTask = [self.manager dataTaskWithRequest:request
                                                     completionHandler:^(NSURLResponse *response, id responseObject, NSError *error) {
        if (error) {
            BLOCK_SAFE_RUN(completion, error, nil);
        } else {
            NSStringEncoding encoding = CFStringConvertEncodingToNSStringEncoding(kCFStringEncodingGB_18030_2000);
            NSString *responseStr = [[NSString alloc] initWithData:responseObject encoding:encoding];
            BLOCK_SAFE_RUN(completion, nil, responseStr);
        }
    }];
    [dataTask resume];
}

- (void)querySinaStockItemsWithNum:(NSString *)stockCode
                              year:(NSString *)year
                           quarter:(NSString *)quarter
                        completion:(StockItemsCompletion) completion {
    [self querySinaStockWithNum:stockCode year:year quarter:quarter completion:^(NSError *error, NSString *html) {
        if (error) {
            BLOCK_SAFE_RUN(completion, error, nil);
        } else {
            dispatch_async(dispatch_get_global_queue(0, 0), ^{
                NSArray *items = [APStockManager parseStockItemsWithSinaHtml:html];
                dispatch_async(dispatch_get_main_queue(), ^{
                    BLOCK_SAFE_RUN(completion, nil, items);
                });
            });
            
        }
    }];
}

#pragma mark - 获取实时股票信息

/**
 获取实时股票信息

 @param codes code以,分割代表多只股票
 @param completion <#completion description#>
 */
- (void) querySinaNowStockInfoWithStockCodes:(NSString *) codes
                                  completion:(StockCompletion) completion{
    NSMutableArray *codeArr = [NSMutableArray array];
    [[codes componentsSeparatedByString:@","] enumerateObjectsUsingBlock:^(NSString * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
        [codeArr addObject:[obj jk_prefixStockCode]];
    }];
    NSString *codesStr = [codeArr componentsJoinedByString:@","];
    NSString *url = [NSString stringWithFormat:@"http://hq.sinajs.cn/list=%@", codesStr];
    dispatch_async(dispatch_get_global_queue(0, 0), ^{
        
        NSData *data = [NSData dataWithContentsOfURL:[NSURL URLWithString:url]];
        NSStringEncoding encoding = CFStringConvertEncodingToNSStringEncoding(kCFStringEncodingGB_18030_2000);
        NSString *responseStr = [[NSString alloc] initWithData:data encoding:encoding];
        if (responseStr.length > 0) {
            NSArray<NSString *> *arr = [responseStr componentsSeparatedByString:@";"];
            NSMutableArray *stocks = [NSMutableArray array];
            [arr enumerateObjectsUsingBlock:^(NSString * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
                NSArray<NSString *> *arr1 = [obj componentsSeparatedByString:@"="];
                if (arr1.count == 2) {
                    NSString *code = arr1[0].jk_stockNumbers.firstObject;
                    NSArray<NSString *> *infos = [[arr1[1] stringByReplacingOccurrencesOfString:@"\"" withString:@""] componentsSeparatedByString:@","];
                    if (infos.count == 33) {
                        APStock *stock = [APStock new];
                        stock.code = code;
                        stock.name = infos[0];
                        stock.todayStartPrice = [infos[1] doubleValue];
                        stock.yestodayEndPrice = [infos[2] doubleValue];
                        stock.todayCurrentPrice = [infos[3] doubleValue];
                        stock.todayMaxPrice = [infos[4] doubleValue];
                        stock.todayMinPrice = [infos[5] doubleValue];
                        stock.todayCompeteBuyPrice = [infos[6] doubleValue];
                        stock.todayCompeteSellPrice = [infos[7] doubleValue];
                        stock.todayTransactionCount = [infos[8] integerValue];
                        stock.todayTransactionMoney = [infos[9] doubleValue];
                        stock.todayCurBuy1Count = [infos[10] integerValue];
                        stock.todayCurBuy1Price = [infos[11] doubleValue];
                        stock.todayCurBuy2Count = [infos[12] integerValue];
                        stock.todayCurBuy2Price = [infos[13] doubleValue];
                        stock.todayCurBuy3Count = [infos[14] integerValue];
                        stock.todayCurBuy3Price = [infos[15] doubleValue];
                        stock.todayCurBuy4Count = [infos[16] integerValue];
                        stock.todayCurBuy4Price = [infos[17] doubleValue];
                        stock.todayCurBuy5Count = [infos[18] integerValue];
                        stock.todayCurBuy5Price = [infos[19] doubleValue];
                        stock.todayCurSell1Count = [infos[20] integerValue];
                        stock.todayCurSell1Price = [infos[21] doubleValue];
                        stock.todayCurSell2Count = [infos[22] integerValue];
                        stock.todayCurSell2Price = [infos[23] doubleValue];
                        stock.todayCurSell3Count = [infos[24] integerValue];
                        stock.todayCurSell3Price = [infos[25] doubleValue];
                        stock.todayCurSell4Count = [infos[26] integerValue];
                        stock.todayCurSell4Price = [infos[27] doubleValue];
                        stock.todayCurSell5Count = [infos[28] integerValue];
                        stock.todayCurSell5Price = [infos[29] doubleValue];
                        stock.todayDate = infos[30];
                        stock.todayTime = infos[31];
                        [stocks addObject:stock];
                    }
                }
            }];
            dispatch_async(dispatch_get_main_queue(), ^{
                BLOCK_SAFE_RUN(completion, nil, stocks);
            });
        } else {
            dispatch_async(dispatch_get_main_queue(), ^{
                BLOCK_SAFE_RUN(completion, [NSError new], nil);
            });
        }
    });
}

- (void) queryTencentNowStockInfoWithStockCodes:(NSString *) codes
                                  completion:(StockCompletion) completion {
    NSMutableArray *codeArr = [NSMutableArray array];
    [[codes componentsSeparatedByString:@","] enumerateObjectsUsingBlock:^(NSString * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
        [codeArr addObject:[obj jk_prefixStockCode]];
    }];
    NSString *codesStr = [codeArr componentsJoinedByString:@","];
    NSString *url = [NSString stringWithFormat:@"http://qt.gtimg.cn/q=%@", codesStr];
    DLog(@"%@",url);
    [[JKHelper sharedInstance] jk_networkRequestProcessGB2312WithUrl:url
                                                          Completion:^(NSError *error, NSString *result) {
                                                              dispatch_async(dispatch_get_global_queue(0, 0), ^{
                                                                  NSMutableArray *stocks = [NSMutableArray array];
                                                                  [[result componentsSeparatedByString:@";"] enumerateObjectsUsingBlock:^(NSString * _Nonnull aString, NSUInteger idx, BOOL * _Nonnull stop) {
                                                                      NSArray<NSString *> *arr = [aString jk_macthWithRegex:@"\"(.+)\""];
                                                                      if (arr.count) {
                                                                          [arr enumerateObjectsUsingBlock:^(NSString * _Nonnull obj, NSUInteger idx, BOOL * _Nonnull stop) {
                                                                              NSString *infos = [obj stringByReplacingOccurrencesOfString:@"\"" withString:@""];
                                                                              NSArray<NSString *> *items = [infos componentsSeparatedByString:@"~"];
                                                                              APStock *stock = [APStock new];
                                                                              stock.name  = items[1];
                                                                              stock.code  = items[2];
                                                                              stock.todayCurrentPrice  = [items[3] doubleValue];
                                                                              stock.yestodayEndPrice  = [items[4] doubleValue];
                                                                              stock.todayStartPrice  = [items[5] doubleValue];
                                                                              stock.todayTransactionCount  = [items[6] integerValue];
                                                                              stock.todayTransactionOutCount  = [items[7] integerValue];
                                                                              stock.todayTransactionInnerCount  = [items[8] integerValue];
                                                                              stock.todayCurBuy1Price  = [items[9] doubleValue];
                                                                              stock.todayCurBuy1Count  = [items[10] integerValue];
                                                                              stock.todayCurBuy2Price  = [items[11] doubleValue];
                                                                              stock.todayCurBuy2Count  = [items[12] integerValue];
                                                                              stock.todayCurBuy3Price  = [items[13] doubleValue];
                                                                              stock.todayCurBuy3Count  = [items[14] integerValue];
                                                                              stock.todayCurBuy4Price  = [items[15] doubleValue];
                                                                              stock.todayCurBuy4Count  = [items[16] integerValue];
                                                                              stock.todayCurBuy5Price  = [items[17] doubleValue];
                                                                              stock.todayCurBuy5Count  = [items[18] integerValue];
                                                                              stock.todayCurSell1Price  = [items[19] doubleValue];
                                                                              stock.todayCurSell1Count  = [items[20] integerValue];
                                                                              stock.todayCurSell2Price  = [items[21] doubleValue];
                                                                              stock.todayCurSell2Count  = [items[22] integerValue];
                                                                              stock.todayCurSell3Price  = [items[23] doubleValue];
                                                                              stock.todayCurSell3Count  = [items[24] integerValue];
                                                                              stock.todayCurSell4Price  = [items[25] doubleValue];
                                                                              stock.todayCurSell4Count  = [items[26] integerValue];
                                                                              stock.todayCurSell5Price  = [items[27] doubleValue];
                                                                              stock.todayCurSell5Count  = [items[28] integerValue];
                                                                              stock.todayRecentDealInfo = items[29];
                                                                              stock.todayDateTime       = items[30];
                                                                              stock.todayUpDownDeltaPrice = [items[31] doubleValue];
                                                                              stock.todayUpDownRate = [items[32] doubleValue];
                                                                              stock.todayMaxPrice = [items[33] doubleValue];
                                                                              stock.todayMinPrice = [items[34] doubleValue];
                                                                              stock.todayTransactionMoney = [items[37] integerValue];
                                                                              stock.todayChangeHandRate = [items[38] doubleValue];
                                                                              stock.priceEarningRatio = [items[39] doubleValue];
                                                                              stock.amplitude = [items[43] doubleValue];
                                                                              stock.totalCirculateMarketValue = [items[44] doubleValue];
                                                                              stock.totalMarketValue = [items[45] doubleValue];
                                                                              stock.priceValueRatio = [items[46] doubleValue];
                                                                              stock.upLimitPrice = [items[47] doubleValue];
                                                                              stock.downLimitPrice = [items[48] doubleValue];
                                                                              [stocks addObject:stock];
                                                                          }];
                                                                      };
                                                                  }];
                                                                  dispatch_async(dispatch_get_main_queue(), ^{
                                                                      BLOCK_SAFE_RUN(completion, nil, stocks);
                                                                  });
                                                              });
                                                          }];
}


@end
`